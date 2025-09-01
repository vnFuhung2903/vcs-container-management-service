package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"

	"github.com/vnFuhung2903/vcs-container-management-service/dto"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
	"github.com/vnFuhung2903/vcs-container-management-service/mocks/docker"
	"github.com/vnFuhung2903/vcs-container-management-service/mocks/interfaces"
	"github.com/vnFuhung2903/vcs-container-management-service/mocks/logger"
	"github.com/vnFuhung2903/vcs-container-management-service/mocks/repositories"
)

type ContainerServiceSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	containerService IContainerService
	mockRepo         *repositories.MockIContainerRepository
	dockerClient     *docker.MockIDockerClient
	redisClient      *interfaces.MockIRedisClient
	logger           *logger.MockILogger
	ctx              context.Context
}

func (s *ContainerServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = repositories.NewMockIContainerRepository(s.ctrl)
	s.dockerClient = docker.NewMockIDockerClient(s.ctrl)
	s.redisClient = interfaces.NewMockIRedisClient(s.ctrl)
	s.logger = logger.NewMockILogger(s.ctrl)
	s.containerService = NewContainerService(s.mockRepo, s.dockerClient, s.redisClient, s.logger)
	s.ctx = context.Background()
}

func (s *ContainerServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestContainerServiceSuite(t *testing.T) {
	suite.Run(t, new(ContainerServiceSuite))
}

func (s *ContainerServiceSuite) TestCreate() {
	containerResp := &container.CreateResponse{ID: "test-id"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "container", "testcontainers/ryuk:0.12.0").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(nil)
	s.mockRepo.EXPECT().Create("test-id", "container", entities.ContainerOn, "127.0.0.1").Return(&entities.Container{
		ContainerId:   "test-id",
		ContainerName: "container",
		Status:        entities.ContainerOn,
		Ipv4:          "127.0.0.1",
	}, nil)
	s.logger.EXPECT().Info("container created successfully", zap.String("containerId", "test-id")).Times(1)

	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.NoError(err)
	s.Equal("container", result.ContainerName)
	s.Equal("test-id", result.ContainerId)
}

func (s *ContainerServiceSuite) TestCreateRedisGetError() {
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(nil, errors.New("redis get error"))
	s.logger.EXPECT().Error("failed to retrieve containers from redis", gomock.Any()).Times(1)
	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.ErrorContains(err, "redis get error")
	s.Nil(result)
}

func (s *ContainerServiceSuite) TestCreateDockerCreateError() {
	existingContainers := []entities.ContainerWithStatus{}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "container", "testcontainers/ryuk:0.12.0").Return(nil, errors.New("docker create error"))
	s.logger.EXPECT().Error("failed to create docker container", gomock.Any()).Times(1)

	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.ErrorContains(err, "docker create error")
	s.Nil(result)
}

func (s *ContainerServiceSuite) TestCreateDockerStartError() {
	containerResp := &container.CreateResponse{ID: "test-id"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOff},
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "container", "testcontainers/ryuk:0.12.0").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(errors.New("docker start error"))
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOff)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("")
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(nil)
	s.mockRepo.EXPECT().Create("test-id", "container", entities.ContainerOff, "").Return(&entities.Container{
		ContainerId:   "test-id",
		ContainerName: "container",
		Status:        entities.ContainerOff,
		Ipv4:          "",
	}, nil)
	s.logger.EXPECT().Error("failed to start docker container", zap.Error(errors.New("docker start error"))).Times(1)
	s.logger.EXPECT().Info("container created successfully", zap.String("containerId", "test-id")).Times(1)

	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.NoError(err)
	s.Equal("container", result.ContainerName)
	s.Equal("test-id", result.ContainerId)
}

func (s *ContainerServiceSuite) TestCreateRepoError() {
	containerResp := &container.CreateResponse{ID: "test-id"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "container", "testcontainers/ryuk:0.12.0").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(nil)
	s.mockRepo.EXPECT().Create("test-id", "container", entities.ContainerOn, "127.0.0.1").Return(nil, errors.New("db error"))
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().Delete(s.ctx, "test-id").Return(nil)
	s.logger.EXPECT().Error("failed to create container", gomock.Any()).Times(1)

	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.ErrorContains(err, "db error")
	s.Nil(result)
}

func (s *ContainerServiceSuite) TestCreateRepoAndDockerStopError() {
	containerResp := &container.CreateResponse{ID: "test-id"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "container", "testcontainers/ryuk:0.12.0").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(nil)
	s.mockRepo.EXPECT().Create("test-id", "container", entities.ContainerOn, "127.0.0.1").Return(nil, errors.New("db error"))
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(errors.New("docker stop error"))
	s.logger.EXPECT().Error("failed to create container", gomock.Any()).Times(1)
	s.logger.EXPECT().Error("failed to stop docker container", gomock.Any()).Times(1)

	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.ErrorContains(err, "docker stop error")
	s.Nil(result)
}

func (s *ContainerServiceSuite) TestCreateRepoAndDockerDeleteError() {
	containerResp := &container.CreateResponse{ID: "test-id"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "container", "testcontainers/ryuk:0.12.0").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(nil)
	s.mockRepo.EXPECT().Create("test-id", "container", entities.ContainerOn, "127.0.0.1").Return(nil, errors.New("db error"))
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().Delete(s.ctx, "test-id").Return(errors.New("docker delete error"))
	s.logger.EXPECT().Error("failed to create container", gomock.Any()).Times(1)
	s.logger.EXPECT().Error("failed to delete docker container", gomock.Any()).Times(1)

	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.ErrorContains(err, "docker delete error")
	s.Nil(result)
}

func (s *ContainerServiceSuite) TestCreateRedisSetError() {
	containerResp := &container.CreateResponse{ID: "test-id"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "container", "testcontainers/ryuk:0.12.0").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(errors.New("redis set error"))
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().Delete(s.ctx, "test-id").Return(nil)
	s.logger.EXPECT().Error("failed to set containers in redis", gomock.Any()).Times(1)

	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.ErrorContains(err, "redis set error")
	s.Nil(result)
}

func (s *ContainerServiceSuite) TestCreateRedisSetAndDockerStopError() {
	containerResp := &container.CreateResponse{ID: "test-id"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "container", "testcontainers/ryuk:0.12.0").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(errors.New("redis set error"))
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(errors.New("docker stop error"))
	s.logger.EXPECT().Error("failed to set containers in redis", gomock.Any()).Times(1)
	s.logger.EXPECT().Error("failed to stop docker container", gomock.Any()).Times(1)

	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.ErrorContains(err, "docker stop error")
	s.Nil(result)
}

func (s *ContainerServiceSuite) TestCreateRedisSetAndDockerDeleteError() {
	containerResp := &container.CreateResponse{ID: "test-id"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "container", "testcontainers/ryuk:0.12.0").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(errors.New("redis set error"))
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().Delete(s.ctx, "test-id").Return(errors.New("docker delete error"))
	s.logger.EXPECT().Error("failed to set containers in redis", gomock.Any()).Times(1)
	s.logger.EXPECT().Error("failed to delete docker container", gomock.Any()).Times(1)

	result, err := s.containerService.Create(s.ctx, "container", "testcontainers/ryuk:0.12.0")
	s.ErrorContains(err, "docker delete error")
	s.Nil(result)
}

func (s *ContainerServiceSuite) TestView() {
	filter := dto.ContainerFilter{}
	sort := dto.ContainerSort{Field: "container_id", Order: "asc"}
	expected := []*entities.Container{{ContainerId: "abc"}}

	s.mockRepo.EXPECT().View(filter, 1, 10, sort).Return(expected, int64(1), nil)
	s.logger.EXPECT().Info("containers listed successfully", gomock.Any()).Times(1)

	result, total, err := s.containerService.View(s.ctx, filter, 1, 10, sort)
	s.NoError(err)
	s.Equal(int64(1), total)
	s.Equal(expected, result)
}

func (s *ContainerServiceSuite) TestViewError() {
	s.mockRepo.EXPECT().View(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), errors.New("db error"))
	s.logger.EXPECT().Error("failed to view containers", gomock.Any()).Times(1)

	_, _, err := s.containerService.View(s.ctx, dto.ContainerFilter{}, 1, 10, dto.ContainerSort{})
	s.ErrorContains(err, "db error")
}

func (s *ContainerServiceSuite) TestViewInvalidRange() {
	s.logger.EXPECT().Error("failed to view containers", gomock.Any()).Times(1)
	_, _, err := s.containerService.View(s.ctx, dto.ContainerFilter{}, 0, 10, dto.ContainerSort{})
	s.ErrorContains(err, "invalid range")
}

func (s *ContainerServiceSuite) TestUpdateOn() {
	updateData := dto.ContainerUpdate{Status: "ON"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}

	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.logger.EXPECT().Warn("container not found in redis", zap.String("containerId", "test-id")).Times(1)
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(nil)
	s.mockRepo.EXPECT().Update("test-id", entities.ContainerOn, "127.0.0.1").Return(nil)
	s.logger.EXPECT().Info("container updated successfully", gomock.Any()).Times(1)

	err := s.containerService.Update(s.ctx, "test-id", updateData)
	s.NoError(err)
}

func (s *ContainerServiceSuite) TestUpdateOff() {
	updateData := dto.ContainerUpdate{Status: "OFF"}
	existingContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOff},
	}

	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOff)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("")
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(nil)
	s.mockRepo.EXPECT().Update("test-id", entities.ContainerOff, "").Return(nil)
	s.logger.EXPECT().Info("container updated successfully", gomock.Any()).Times(1)

	err := s.containerService.Update(s.ctx, "test-id", updateData)
	s.NoError(err)
}

func (s *ContainerServiceSuite) TestUpdateInvalidStatus() {
	updateData := dto.ContainerUpdate{Status: "INVALID"}

	err := s.containerService.Update(s.ctx, "test-id", updateData)
	s.ErrorContains(err, "invalid status")
}

func (s *ContainerServiceSuite) TestUpdateDockerStartError() {
	updateData := dto.ContainerUpdate{Status: "ON"}

	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(errors.New("docker start error"))
	s.logger.EXPECT().Error("failed to start docker container", gomock.Any()).Times(1)

	err := s.containerService.Update(s.ctx, "test-id", updateData)
	s.ErrorContains(err, "docker start error")
}

func (s *ContainerServiceSuite) TestUpdateDockerStopError() {
	updateData := dto.ContainerUpdate{Status: "OFF"}

	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(errors.New("docker stop error"))
	s.logger.EXPECT().Error("failed to stop docker container", gomock.Any()).Times(1)

	err := s.containerService.Update(s.ctx, "test-id", updateData)
	s.ErrorContains(err, "docker stop error")
}

func (s *ContainerServiceSuite) TestUpdateRedisGetError() {
	updateData := dto.ContainerUpdate{Status: "ON"}
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(nil, errors.New("redis get error"))
	s.logger.EXPECT().Error("failed to retrieve containers from redis", gomock.Any()).Times(1)

	err := s.containerService.Update(s.ctx, "test-id", updateData)
	s.ErrorContains(err, "redis get error")
}

func (s *ContainerServiceSuite) TestUpdateRedisSetError() {
	updateData := dto.ContainerUpdate{Status: "OFF"}
	existingContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOff},
	}

	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOff)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("")
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(errors.New("redis set error"))
	s.logger.EXPECT().Error("failed to update containers in redis", gomock.Any()).Times(1)

	err := s.containerService.Update(s.ctx, "test-id", updateData)
	s.ErrorContains(err, "redis set error")
}

func (s *ContainerServiceSuite) TestUpdateRepoError() {
	updateData := dto.ContainerUpdate{Status: "OFF"}
	existingContainers := []entities.ContainerWithStatus{}
	updatedContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOff},
	}

	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOff)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("")
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.logger.EXPECT().Warn("container not found in redis", zap.String("containerId", "test-id")).Times(1)
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(nil)
	s.mockRepo.EXPECT().Update("test-id", entities.ContainerOff, "").Return(errors.New("update failed"))
	s.logger.EXPECT().Error("failed to update container", gomock.Any()).Times(1)

	err := s.containerService.Update(s.ctx, "test-id", updateData)
	s.ErrorContains(err, "update failed")
}

func (s *ContainerServiceSuite) TestDelete() {
	existingContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}
	updatedContainers := []entities.ContainerWithStatus{}

	s.mockRepo.EXPECT().Delete("test-id").Return(nil)
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedContainers).Return(nil)
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().Delete(s.ctx, "test-id").Return(nil)
	s.logger.EXPECT().Info("container deleted successfully", zap.String("containerId", "test-id")).Times(1)

	err := s.containerService.Delete(s.ctx, "test-id")
	s.NoError(err)
}

func (s *ContainerServiceSuite) TestDeleteDockerStopError() {
	existingContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}
	s.mockRepo.EXPECT().Delete("test-id").Return(nil)
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", gomock.Any()).Return(nil)
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(errors.New("stop failed"))
	s.logger.EXPECT().Error("failed to stop docker container", gomock.Any()).Times(1)

	err := s.containerService.Delete(s.ctx, "test-id")
	s.ErrorContains(err, "stop failed")
}

func (s *ContainerServiceSuite) TestDeleteDockerDeleteError() {
	existingContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}
	s.mockRepo.EXPECT().Delete("test-id").Return(nil)
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", gomock.Any()).Return(nil)
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().Delete(s.ctx, "test-id").Return(errors.New("delete failed"))
	s.logger.EXPECT().Error("failed to delete docker container", gomock.Any()).Times(1)

	err := s.containerService.Delete(s.ctx, "test-id")
	s.ErrorContains(err, "delete failed")
}

func (s *ContainerServiceSuite) TestDeleteRepoError() {
	s.mockRepo.EXPECT().Delete("test-id").Return(errors.New("delete failed"))
	s.logger.EXPECT().Error("failed to delete container", gomock.Any()).Times(1)

	err := s.containerService.Delete(s.ctx, "test-id")
	s.ErrorContains(err, "delete failed")
}

func (s *ContainerServiceSuite) TestDeleteRedisGetError() {
	s.mockRepo.EXPECT().Delete("test-id").Return(nil)
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(nil, errors.New("redis get error"))
	s.logger.EXPECT().Error("failed to retrieve containers from redis", gomock.Any()).Times(1)

	err := s.containerService.Delete(s.ctx, "test-id")
	s.ErrorContains(err, "redis get error")
}

func (s *ContainerServiceSuite) TestDeleteRedisSetError() {
	existingContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}
	s.mockRepo.EXPECT().Delete("test-id").Return(nil)
	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingContainers, nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", gomock.Any()).Return(errors.New("redis set error"))
	s.logger.EXPECT().Error("failed to update containers in redis", gomock.Any()).Times(1)

	err := s.containerService.Delete(s.ctx, "test-id")
	s.ErrorContains(err, "redis set error")
}

func (s *ContainerServiceSuite) TestImport() {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", "Container Name")
	f.SetCellValue(sheet, "B1", "Image Name")

	f.SetCellValue(sheet, "A2", "test-name")
	f.SetCellValue(sheet, "B2", "nginx")

	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())
	file := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	containerResp := &container.CreateResponse{ID: "test-id"}
	containerEntity := &entities.Container{
		ContainerId:   "test-id",
		ContainerName: "test-name",
		Status:        entities.ContainerOn,
		Ipv4:          "127.0.0.1",
	}
	containers := []*entities.Container{containerEntity}
	existingRedisContainers := []entities.ContainerWithStatus{}
	updatedRedisContainers := []entities.ContainerWithStatus{
		{ContainerId: "test-id", Status: entities.ContainerOn},
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingRedisContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "test-name", "nginx").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.mockRepo.EXPECT().CreateInBatches(containers).Return(nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", updatedRedisContainers).Return(nil)
	s.logger.EXPECT().Info("containers imported successfully").Times(1)

	resp, err := s.containerService.Import(s.ctx, file)
	s.NoError(err)
	s.Equal(1, resp.SuccessCount)
	s.Equal(0, resp.FailedCount)
	s.Contains(resp.SuccessContainers, "test-name")
}

func (s *ContainerServiceSuite) TestImportInvalidExcelFile() {
	data := []byte("this is not a real Excel file")
	reader := bytes.NewReader(data)

	fakeFile := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	s.logger.EXPECT().Error(gomock.Any(), gomock.Any()).Times(1)

	resp, err := s.containerService.Import(s.ctx, fakeFile)
	s.Error(err)
	s.Nil(resp)
}

func (s *ContainerServiceSuite) TestImportWithMissingHeaderRows() {
	existingRedisContainers := []entities.ContainerWithStatus{}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingRedisContainers, nil)
	s.logger.EXPECT().Error("failed to import containers", gomock.Any()).Times(1)

	f := excelize.NewFile()
	sheetName := "Sheet1"
	f.SetSheetName("Sheet1", sheetName)
	f.SetCellValue(sheetName, "A1", "Container Id")

	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())

	fakeFile := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	resp, err := s.containerService.Import(s.ctx, fakeFile)
	s.Error(err)
	s.Nil(resp)
}

func (s *ContainerServiceSuite) TestImportWithInvalidHeaderRows() {
	existingRedisContainers := []entities.ContainerWithStatus{}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingRedisContainers, nil)
	s.logger.EXPECT().Error("failed to import containers", gomock.Any()).Times(1)

	f := excelize.NewFile()
	sheetName := "Sheet1"
	f.SetSheetName("Sheet1", sheetName)
	f.SetCellValue(sheetName, "A1", "Container Id")
	f.SetCellValue(sheetName, "B1", "Image Name")

	f.SetCellValue(sheetName, "A2", "test-id")

	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())

	fakeFile := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	resp, err := s.containerService.Import(s.ctx, fakeFile)
	s.Error(err)
	s.Nil(resp)
}

func (s *ContainerServiceSuite) TestImportWithInvalidRows() {
	existingRedisContainers := []entities.ContainerWithStatus{}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingRedisContainers, nil)
	s.logger.EXPECT().Warn("skipping invalid row", gomock.Any()).Times(1)
	s.logger.EXPECT().Info("containers imported successfully").Times(1)

	f := excelize.NewFile()
	sheetName := "Sheet1"
	f.SetSheetName("Sheet1", sheetName)
	f.SetCellValue(sheetName, "A1", "Container Name")
	f.SetCellValue(sheetName, "B1", "Image Name")

	f.SetCellValue(sheetName, "A2", "test-name")

	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())

	fakeFile := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	s.mockRepo.EXPECT().CreateInBatches([]*entities.Container{}).Return(nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", existingRedisContainers).Return(nil)
	resp, err := s.containerService.Import(s.ctx, fakeFile)
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(0, resp.SuccessCount)
	s.Equal(0, resp.FailedCount)
}

func (s *ContainerServiceSuite) TestImportRedisGetError() {
	f := excelize.NewFile()
	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())
	file := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(nil, errors.New("redis get error"))
	s.logger.EXPECT().Error("failed to retrieve containers from redis", gomock.Any()).Times(1)

	resp, err := s.containerService.Import(s.ctx, file)
	s.ErrorContains(err, "redis get error")
	s.Nil(resp)
}

func (s *ContainerServiceSuite) TestImportRedisSetError() {
	existingRedisContainers := []entities.ContainerWithStatus{}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingRedisContainers, nil)
	s.logger.EXPECT().Info("containers imported successfully").Times(1)

	f := excelize.NewFile()
	sheetName := "Sheet1"
	f.SetSheetName("Sheet1", sheetName)
	f.SetCellValue(sheetName, "A1", "Container Name")
	f.SetCellValue(sheetName, "B1", "Image Name")

	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())

	fakeFile := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	s.mockRepo.EXPECT().CreateInBatches([]*entities.Container{}).Return(nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", existingRedisContainers).Return(errors.New("redis set error"))
	s.logger.EXPECT().Error("failed to update containers in redis", gomock.Any()).Times(1)
	resp, err := s.containerService.Import(s.ctx, fakeFile)
	s.ErrorContains(err, "redis set error")
	s.Nil(resp)
}

func (s *ContainerServiceSuite) TestImportDockerCreateError() {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", "Container Name")
	f.SetCellValue(sheet, "B1", "Image Name")

	f.SetCellValue(sheet, "A2", "test-name")
	f.SetCellValue(sheet, "B2", "nginx")

	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())
	file := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	existingRedisContainers := []entities.ContainerWithStatus{}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingRedisContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "test-name", "nginx").Return(nil, errors.New("create error"))
	s.mockRepo.EXPECT().CreateInBatches([]*entities.Container{}).Return(nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", existingRedisContainers).Return(nil)
	s.logger.EXPECT().Info("containers imported successfully").Times(1)

	resp, err := s.containerService.Import(s.ctx, file)
	s.NoError(err)
	s.Equal(0, resp.SuccessCount)
	s.Equal(1, resp.FailedCount)
	s.Contains(resp.FailedContainers, "test-name")
}

func (s *ContainerServiceSuite) TestImportInvalidContainerField() {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", "Container Name")
	f.SetCellValue(sheet, "B1", "Image Name")

	f.SetCellValue(sheet, "A2", "")
	f.SetCellValue(sheet, "B2", "nginx")

	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())
	file := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	existingRedisContainers := []entities.ContainerWithStatus{}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingRedisContainers, nil)
	s.mockRepo.EXPECT().CreateInBatches([]*entities.Container{}).Return(nil)
	s.redisClient.EXPECT().Set(s.ctx, "containers", existingRedisContainers).Return(nil)
	s.logger.EXPECT().Info("containers imported successfully").Times(1)

	resp, err := s.containerService.Import(s.ctx, file)
	s.NoError(err)
	s.Equal(0, resp.SuccessCount)
	s.Equal(1, resp.FailedCount)
	s.Contains(resp.FailedContainers, "")
}

func (s *ContainerServiceSuite) TestImportRepoAndDockerStopError() {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", "Container Name")
	f.SetCellValue(sheet, "B1", "Image Name")

	f.SetCellValue(sheet, "A2", "test-name")
	f.SetCellValue(sheet, "B2", "nginx")

	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())
	file := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	containerResp := &container.CreateResponse{ID: "test-id"}
	containerEntity := &entities.Container{
		ContainerId:   "test-id",
		ContainerName: "test-name",
		Status:        entities.ContainerOn,
		Ipv4:          "127.0.0.1",
	}
	containers := []*entities.Container{containerEntity}
	existingRedisContainers := []entities.ContainerWithStatus{}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingRedisContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "test-name", "nginx").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.mockRepo.EXPECT().CreateInBatches(containers).Return(errors.New("db error"))
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(errors.New("docker stop error"))
	s.redisClient.EXPECT().Set(s.ctx, "containers", existingRedisContainers).Return(nil)
	s.logger.EXPECT().Error("failed to stop docker container", zap.String("container_id", containerEntity.ContainerId), gomock.Any()).Times(1)

	resp, err := s.containerService.Import(s.ctx, file)
	s.NoError(err)
	s.Equal(0, resp.SuccessCount)
	s.Equal(1, resp.FailedCount)
	s.Contains(resp.FailedContainers, "test-name")
}

func (s *ContainerServiceSuite) TestImportRepoAndDockerDeleteError() {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", "Container Name")
	f.SetCellValue(sheet, "B1", "Image Name")

	f.SetCellValue(sheet, "A2", "test-name")
	f.SetCellValue(sheet, "B2", "nginx")

	var buf bytes.Buffer
	err := f.Write(&buf)
	s.Require().NoError(err)

	reader := bytes.NewReader(buf.Bytes())
	file := struct {
		io.Reader
		io.ReaderAt
		io.Seeker
		io.Closer
	}{
		Reader:   reader,
		ReaderAt: reader,
		Seeker:   reader,
		Closer:   io.NopCloser(nil),
	}

	containerResp := &container.CreateResponse{ID: "test-id"}
	containerEntity := &entities.Container{
		ContainerId:   "test-id",
		ContainerName: "test-name",
		Status:        entities.ContainerOn,
		Ipv4:          "127.0.0.1",
	}
	containers := []*entities.Container{containerEntity}
	existingRedisContainers := []entities.ContainerWithStatus{}

	s.redisClient.EXPECT().Get(s.ctx, "containers").Return(existingRedisContainers, nil)
	s.dockerClient.EXPECT().Create(s.ctx, "test-name", "nginx").Return(containerResp, nil)
	s.dockerClient.EXPECT().Start(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().GetStatus(s.ctx, "test-id").Return(entities.ContainerOn)
	s.dockerClient.EXPECT().GetIpv4(s.ctx, "test-id").Return("127.0.0.1")
	s.mockRepo.EXPECT().CreateInBatches(containers).Return(errors.New("db error"))
	s.dockerClient.EXPECT().Stop(s.ctx, "test-id").Return(nil)
	s.dockerClient.EXPECT().Delete(s.ctx, "test-id").Return(errors.New("docker stop error"))
	s.redisClient.EXPECT().Set(s.ctx, "containers", existingRedisContainers).Return(nil)
	s.logger.EXPECT().Error("failed to delete docker container", zap.String("container_id", containerEntity.ContainerId), gomock.Any()).Times(1)

	resp, err := s.containerService.Import(s.ctx, file)
	s.NoError(err)
	s.Equal(0, resp.SuccessCount)
	s.Equal(1, resp.FailedCount)
	s.Contains(resp.FailedContainers, "test-name")
}

func (s *ContainerServiceSuite) TestExport() {
	filter := dto.ContainerFilter{}
	sort := dto.ContainerSort{Field: "container_id", Order: "asc"}
	from, to := 1, 10
	containers := []*entities.Container{
		{
			ContainerId:   "abc",
			ContainerName: "test-id",
			Status:        "ON",
			Ipv4:          "192.168.1.1",
			CreatedAt:     time.Now(),
		},
	}

	s.mockRepo.EXPECT().View(filter, from, to-from+1, sort).Return(containers, int64(len(containers)), nil)
	s.logger.EXPECT().Info("containers exported successfully").Times(1)

	result, err := s.containerService.Export(s.ctx, filter, from, to, sort)
	s.NoError(err)
	s.True(len(result) > 0)
}

func (s *ContainerServiceSuite) TestExportInvalidRange() {
	s.logger.EXPECT().Error("failed to export containers", gomock.Any()).Times(1)
	_, err := s.containerService.Export(s.ctx, dto.ContainerFilter{}, 0, 10, dto.ContainerSort{})
	s.ErrorContains(err, "invalid range")
}

func (s *ContainerServiceSuite) TestExportError() {
	filter := dto.ContainerFilter{}
	sort := dto.ContainerSort{Field: "container_id", Order: "asc"}
	from, to := 1, 5

	s.mockRepo.EXPECT().View(filter, from, to-from+1, sort).Return(nil, int64(0), errors.New("fetch error"))

	_, err := s.containerService.Export(s.ctx, filter, from, to, sort)
	s.ErrorContains(err, "fetch error")
}
