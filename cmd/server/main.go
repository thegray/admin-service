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
	"admin-service/internal/domain/example"
	exampleRepo "admin-service/internal/domain/example/repository"
	"admin-service/internal/domain/users"
	usersRepo "admin-service/internal/domain/users/repository"
	"admin-service/pkg/config"
	"admin-service/pkg/logger"
	"admin-service/pkg/middleware"
	"admin-service/pkg/postgres"
	"admin-service/pkg/prometheus"
	serverpkg "admin-service/pkg/server"
)

func main() {
	cfg, err := config.Load(context.Background())
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

	db, err := postgres.Connect(context.Background(), cfg)
	if err != nil {
		log.Fatal("unable to connect to postgres", zap.Error(err))
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
	userService := users.NewService(userRepository, log)

	exampleRepository := exampleRepo.NewInMemoryRepository(log)
	service := example.NewService(exampleRepository, log)
	limiter := middleware.RateLimitMiddleware(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)
	handler := rest.NewHandler(service, userService, log, limiter)
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
