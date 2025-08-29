package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"github.com/vnFuhung2903/vcs-container-management-service/api"
	_ "github.com/vnFuhung2903/vcs-container-management-service/docs"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
	"github.com/vnFuhung2903/vcs-container-management-service/infrastructures/databases"
	"github.com/vnFuhung2903/vcs-container-management-service/infrastructures/messages"
	"github.com/vnFuhung2903/vcs-container-management-service/interfaces"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/docker"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/env"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/logger"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/middlewares"
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/repositories"
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/services"
	"go.uber.org/zap"
)

// @title VCS SMS API
// @version 1.0
// @description Container Management System API
// @host localhost:8081
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	ctx, cancel := context.WithCancel(context.Background())

	env, err := env.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to retrieve env: %v", err)
	}

	logger, err := logger.LoadLogger(env.LoggerEnv)
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}

	postgresDb, err := databases.ConnectPostgresDb(env.PostgresEnv)
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}
	postgresDb.AutoMigrate(&entities.Container{})

	redisRawClient := databases.NewRedisFactory(env.RedisEnv).ConnectRedis()
	defer redisRawClient.Close()
	redisClient := interfaces.NewRedisClient(redisRawClient)

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}
	jwtMiddleware := middlewares.NewJWTMiddleware(env.AuthEnv)

	containerRepository := repositories.NewContainerRepository(postgresDb)
	containerService := services.NewContainerService(containerRepository, dockerClient, redisClient, logger)
	containerHandler := api.NewContainerHandler(containerService, jwtMiddleware)

	kafkaReader, err := messages.NewKafkaFactory(env.KafkaEnv).ConnectKafkaReader("healthcheck")
	if err != nil {
		log.Fatalf("Failed to create kafka reader: %v", err)
	}
	defer kafkaReader.Close()
	kafkaConsumer := interfaces.NewKafkaConsumer(kafkaReader, redisClient, containerRepository, logger)

	r := gin.Default()
	containerHandler.SetupRoutes(r)
	r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))

	go func() {
		for {
			if err := kafkaConsumer.Consume(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				logger.Error("Failed to consume message", zap.Error(err))
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr:    ":8081",
		Handler: r,
	}

	go func() {
		<-quit
		cancel()

		shutdownCtx, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelTimeout()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP server shutdown failed", zap.Error(err))
		}
		logger.Info("Container management service stopped gracefully")
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to run service: %v", err)
	}
}
