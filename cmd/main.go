package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
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
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/workers"
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
	kafkaConsumer := interfaces.NewKafkaConsumer(kafkaReader)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://container.localhost", "http://swagger.localhost"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	containerHandler.SetupRoutes(r)
	r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	consumerWorker := workers.NewConsumerWorker(kafkaConsumer, redisClient, containerRepository, logger)
	consumerWorker.Start()
	defer consumerWorker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	server := &http.Server{
		Addr:    ":8081",
		Handler: r,
	}

	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("HTTP server shutdown failed", zap.Error(err))
		}
		logger.Info("Container management service stopped gracefully")
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to run service: %v", err)
	}
}
