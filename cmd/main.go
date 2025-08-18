package main

import (
	"log"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"github.com/vnFuhung2903/vcs-container-management-service/api"
	_ "github.com/vnFuhung2903/vcs-container-management-service/docs"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
	"github.com/vnFuhung2903/vcs-container-management-service/infrastructures/databases"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/docker"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/env"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/logger"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/middlewares"
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/repositories"
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/services"
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

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		log.Fatalf("Failed to create docker client: %v", err)
	}
	jwtMiddleware := middlewares.NewJWTMiddleware(env.AuthEnv)

	containerRepository := repositories.NewContainerRepository(postgresDb)
	containerService := services.NewContainerService(containerRepository, dockerClient, logger)
	containerHandler := api.NewContainerHandler(containerService, jwtMiddleware)

	r := gin.Default()
	containerHandler.SetupRoutes(r)
	r.GET("/swagger/*any", swagger.WrapHandler(swaggerFiles.Handler))

	if err := r.Run(":8081"); err != nil {
		log.Fatalf("Failed to run service: %v", err)
	} else {
		logger.Info("Container management service is running on port 8081")
	}
}
