package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"admin-service/api/rest"
	auditdomain "admin-service/internal/domain/audit"
	auditRepo "admin-service/internal/domain/audit/repository"
	authdomain "admin-service/internal/domain/auth"
	authrepo "admin-service/internal/domain/auth/repository"
	"admin-service/internal/domain/example"
	exampleRepo "admin-service/internal/domain/example/repository"
	"admin-service/internal/domain/threats"
	threatsRepo "admin-service/internal/domain/threats/repository"
	"admin-service/internal/domain/users"
	usersRepo "admin-service/internal/domain/users/repository"
	"admin-service/pkg/auth"
	"admin-service/pkg/config"
	initpkg "admin-service/pkg/init"
	"admin-service/pkg/logger"
	"admin-service/pkg/middleware"
	"admin-service/pkg/postgres"
	"admin-service/pkg/prometheus"
	"admin-service/pkg/redisclient"
	serverpkg "admin-service/pkg/server"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to load configuration: %v\n", err)
		os.Exit(1)
	}

	loggerCfg := logger.Config{
		Level: cfg.LogLevel,
	}
	log, err := logger.New(loggerCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to construct logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = log.Sync()
	}()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.TraceMiddleware(log))

	metrics := prometheus.NewMetrics()
	router.Use(metrics.Middleware())
	router.GET("/metrics", metrics.Handler())

	db, err := postgres.Connect(ctx, cfg)
	if err != nil {
		log.Fatal("unable to connect to postgres", zap.Error(err))
	}
	if err := postgres.Migrate(db); err != nil {
		log.Fatal("unable to migrate postgres schema", zap.Error(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Warn("unable to access sql.DB for closing", zap.Error(err))
	} else {
		defer func() {
			_ = sqlDB.Close()
		}()
	}

	userRepository := usersRepo.NewPostgresRepository(db, log)
	threatRepository := threatsRepo.NewPostgresRepository(db, log)
	auditRepository := auditRepo.NewPostgresRepository(db, log)
	auditService := auditdomain.NewService(auditRepository, log)

	userService := users.NewService(userRepository, auditService, log)
	if err := initpkg.InitAdmin(ctx, cfg, db, userRepository, userService, log); err != nil {
		log.Fatal("failed to seed admin data", zap.Error(err))
	}

	threatService := threats.NewService(threatRepository, auditService, log)

	redisClient := redisclient.New(cfg)
	defer func() {
		_ = redisClient.Close()
	}()

	tokenManager, err := auth.NewTokenManager(cfg.TokenSecret, cfg.AccessTokenTTL)
	if err != nil {
		log.Fatal("failed to build token manager", zap.Error(err))
	}

	store := authrepo.NewRedisStore(redisClient)
	sessionCache := authdomain.NewSessionCache(store, cfg.UserCacheTTL)

	authService, err := authdomain.NewService(userRepository, tokenManager, sessionCache, store, cfg.RefreshTokenTTL, auditService, log)
	if err != nil {
		log.Fatal("failed to initialize auth service", zap.Error(err))
	}

	authMiddleware := middleware.AuthMiddleware(tokenManager, userRepository, sessionCache, log)

	exampleRepository := exampleRepo.NewInMemoryRepository(log)
	service := example.NewService(exampleRepository, log)
	limiter := middleware.RateLimitMiddleware(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)
	handler := rest.NewHandler(service, userService, threatService, authService, auditService, log, limiter, authMiddleware)
	handler.RegisterRoutes(router)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	log.Info("starting server",
		zap.String("addr", server.Addr),
		zap.String("environment", cfg.Environment),
		zap.Bool("secret_manager", cfg.UseSecretMgr),
	)

	if err := serverpkg.StartHTTPServer(server, log, cfg.ShutdownTimeout); err != nil {
		log.Fatal("server error", zap.Error(err))
	}
}
