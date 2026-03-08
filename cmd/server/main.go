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
	"admin-service/pkg/config"
	"admin-service/pkg/logger"
	"admin-service/pkg/middleware"
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
	defer log.Sync()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.TraceMiddleware(log))

	metrics := prometheus.NewMetrics()
	router.Use(metrics.Middleware())
	router.GET("/metrics", metrics.Handler())

	repo := exampleRepo.NewInMemoryRepository(log)
	service := example.NewService(repo, log)
	limiter := middleware.RateLimitMiddleware(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)
	handler := rest.NewHandler(service, log, limiter)
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
