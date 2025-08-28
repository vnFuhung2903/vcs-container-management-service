package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"slices"

	"github.com/containerd/errdefs"
	"github.com/vnFuhung2903/vcs-container-management-service/dto"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
	"github.com/vnFuhung2903/vcs-container-management-service/interfaces"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/docker"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/logger"
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/repositories"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

type IContainerService interface {
	Create(ctx context.Context, containerName string, imageName string) (*entities.Container, error)
	View(ctx context.Context, containerFilter dto.ContainerFilter, from int, to int, sort dto.ContainerSort) ([]*entities.Container, int64, error)
	Update(ctx context.Context, containerId string, updateData dto.ContainerUpdate) error
	Import(ctx context.Context, file multipart.File) (*dto.ImportResponse, error)
	Export(ctx context.Context, filter dto.ContainerFilter, from int, to int, sort dto.ContainerSort) ([]byte, error)
	Delete(ctx context.Context, containerId string) error
}

type containerService struct {
	containerRepo repositories.IContainerRepository
	dockerClient  docker.IDockerClient
	redisClient   interfaces.IRedisClient
	logger        logger.ILogger
}

func NewContainerService(repo repositories.IContainerRepository, dockerClient docker.IDockerClient, redisClient interfaces.IRedisClient, logger logger.ILogger) IContainerService {
	return &containerService{
		containerRepo: repo,
		dockerClient:  dockerClient,
		redisClient:   redisClient,
		logger:        logger,
	}
}

func (s *containerService) Create(ctx context.Context, containerName string, imageName string) (*entities.Container, error) {
	containers, err := s.redisClient.Get(ctx, "containers")
	if err != nil {
		s.logger.Error("failed to retrieve containers from redis", zap.Error(err))
		return nil, err
	}

	con, err := s.dockerClient.Create(ctx, containerName, imageName)
	if err != nil {
		s.logger.Error("failed to create docker container", zap.Error(err))
		return nil, err
	}

	if err := s.dockerClient.Start(ctx, con.ID); err != nil {
		s.logger.Error("failed to start docker container", zap.Error(err))
	}

	status := s.dockerClient.GetStatus(ctx, con.ID)
	ipv4 := s.dockerClient.GetIpv4(ctx, con.ID)

	containers = slices.Insert(containers, len(containers), entities.ContainerWithStatus{
		ContainerId: con.ID,
		Status:      status,
	})

	if err := s.redisClient.Set(ctx, "containers", containers); err != nil {
		s.logger.Error("failed to set containers in redis", zap.Error(err))
		if err := s.dockerClient.Stop(ctx, con.ID); err != nil {
			s.logger.Error("failed to stop docker container", zap.Error(err))
			return nil, err
		}
		if err := s.dockerClient.Delete(ctx, con.ID); err != nil {
			s.logger.Error("failed to delete docker container", zap.Error(err))
			return nil, err
		}
		return nil, err
	}

	container, err := s.containerRepo.Create(con.ID, containerName, status, ipv4)
	if err != nil {
		s.logger.Error("failed to create container", zap.Error(err))
		if err := s.dockerClient.Stop(ctx, con.ID); err != nil {
			s.logger.Error("failed to stop docker container", zap.Error(err))
			return nil, err
		}
		if err := s.dockerClient.Delete(ctx, con.ID); err != nil {
			s.logger.Error("failed to delete docker container", zap.Error(err))
			return nil, err
		}
		return nil, err
	}

	s.logger.Info("container created successfully", zap.String("containerId", con.ID))
	return container, nil
}

func (s *containerService) View(ctx context.Context, filter dto.ContainerFilter, from int, to int, sort dto.ContainerSort) ([]*entities.Container, int64, error) {
	if from < 1 {
		err := errors.New("invalid range")
		s.logger.Error("failed to view containers", zap.Error(err))
		return nil, 0, err
	}
	limit := max(to-from+1, -1)

	containers, total, err := s.containerRepo.View(filter, from, limit, sort)
	if err != nil {
		s.logger.Error("failed to view containers", zap.Error(err))
		return nil, 0, err
	}

	s.logger.Info("containers listed successfully", zap.Int("count", int(total)))
	return containers, total, nil
}

func (s *containerService) Update(ctx context.Context, containerId string, updateData dto.ContainerUpdate) error {
	if updateData.Status != entities.ContainerOn && updateData.Status != entities.ContainerOff {
		return fmt.Errorf("invalid status: %s", updateData.Status)
	}

	if updateData.Status == entities.ContainerOn {
		if err := s.dockerClient.Start(ctx, containerId); err != nil {
			s.logger.Error("failed to start docker container", zap.Error(err))
			return err
		}
	} else {
		if err := s.dockerClient.Stop(ctx, containerId); err != nil {
			s.logger.Error("failed to stop docker container", zap.Error(err))
			return err
		}
	}

	status := s.dockerClient.GetStatus(ctx, containerId)
	ipv4 := s.dockerClient.GetIpv4(ctx, containerId)

	containers, err := s.redisClient.Get(ctx, "containers")
	if err != nil {
		s.logger.Error("failed to retrieve containers from redis", zap.Error(err))
		return err
	}

	index := slices.IndexFunc(containers, func(c entities.ContainerWithStatus) bool {
		return c.ContainerId == containerId
	})
	if index == -1 {
		s.logger.Warn("container not found in redis", zap.String("containerId", containerId))
		containers = slices.Insert(containers, len(containers), entities.ContainerWithStatus{
			ContainerId: containerId,
			Status:      status,
		})
		index = len(containers) - 1
	}
	containers[index].Status = status

	if err := s.redisClient.Set(ctx, "containers", containers); err != nil {
		s.logger.Error("failed to update containers in redis", zap.Error(err))
		return err
	}

	if err := s.containerRepo.Update(containerId, status, ipv4); err != nil {
		s.logger.Error("failed to update container", zap.Error(err))
		return err
	}
	s.logger.Info("container updated successfully", zap.String("containerId", containerId))
	return nil
}

func (s *containerService) Delete(ctx context.Context, containerId string) error {
	if err := s.containerRepo.Delete(containerId); err != nil {
		s.logger.Error("failed to delete container", zap.Error(err))
		return err
	}

	containers, err := s.redisClient.Get(ctx, "containers")
	if err != nil {
		s.logger.Error("failed to retrieve containers from redis", zap.Error(err))
		return err
	}

	containers = slices.DeleteFunc(containers, func(c entities.ContainerWithStatus) bool {
		return c.ContainerId == containerId
	})
	containers = slices.Clip(containers)

	if err := s.redisClient.Set(ctx, "containers", containers); err != nil {
		s.logger.Error("failed to update containers in redis", zap.Error(err))
		return err
	}

	if err := s.dockerClient.Stop(ctx, containerId); err != nil && !errdefs.IsNotFound(err) {
		s.logger.Error("failed to stop docker container", zap.Error(err))
		return err
	}

	if err := s.dockerClient.Delete(ctx, containerId); err != nil && !errdefs.IsNotFound(err) {
		s.logger.Error("failed to delete docker container", zap.Error(err))
		return err
	}

	s.logger.Info("container deleted successfully", zap.String("containerId", containerId))
	return nil
}

func (s *containerService) Import(ctx context.Context, file multipart.File) (*dto.ImportResponse, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		s.logger.Error("failed to import containers", zap.Error(err))
		return nil, err
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		s.logger.Error("failed to import containers", zap.String("sheetName", sheetName), zap.Error(err))
		return nil, err
	}

	result := &dto.ImportResponse{}
	containers := make([]*entities.Container, 0)

	redisContainers, err := s.redisClient.Get(ctx, "containers")
	if err != nil {
		s.logger.Error("failed to retrieve containers from redis", zap.Error(err))
		return nil, err
	}

	for i, row := range rows {
		if i == 0 {
			if len(row) < 2 {
				err := errors.New("invalid header row")
				s.logger.Error("failed to import containers", zap.Error(err))
				return nil, err
			}
			containerName := strings.TrimSpace(row[0])
			imageName := strings.TrimSpace(row[1])
			if containerName != "Container Name" || imageName != "Image Name" {
				err := errors.New("invalid header row")
				s.logger.Error("failed to import containers", zap.Error(err))
				return nil, err
			}
			continue
		}
		if len(row) < 2 {
			s.logger.Warn("skipping invalid row", zap.Int("row", i+1))
			continue
		}

		containerName := strings.TrimSpace(row[0])
		imageName := strings.TrimSpace(row[1])

		if containerName == "" || imageName == "" {
			result.FailedCount++
			result.FailedContainers = append(result.FailedContainers, containerName)
			continue
		}

		con, err := s.dockerClient.Create(ctx, containerName, imageName)
		if err != nil {
			result.FailedCount++
			result.FailedContainers = append(result.FailedContainers, containerName)
			continue
		}

		s.dockerClient.Start(ctx, con.ID)
		status := s.dockerClient.GetStatus(ctx, con.ID)
		ipv4 := s.dockerClient.GetIpv4(ctx, con.ID)

		redisContainers = slices.Insert(redisContainers, len(redisContainers), entities.ContainerWithStatus{
			ContainerId: con.ID,
			Status:      status,
		})

		containers = append(containers, &entities.Container{
			ContainerId:   con.ID,
			ContainerName: containerName,
			Status:        status,
			Ipv4:          ipv4,
		})
	}

	if err := s.containerRepo.CreateInBatches(containers); err != nil {
		result.FailedCount += len(containers)
		for _, container := range containers {
			result.FailedContainers = append(result.FailedContainers, container.ContainerName)
			redisContainers = slices.DeleteFunc(redisContainers, func(c entities.ContainerWithStatus) bool {
				return c.ContainerId == container.ContainerId
			})
			containers = slices.Clip(containers)
			if err := s.dockerClient.Stop(ctx, container.ContainerId); err != nil {
				s.logger.Error("failed to stop docker container", zap.String("container_id", container.ContainerId), zap.Error(err))
			} else if err := s.dockerClient.Delete(ctx, container.ContainerId); err != nil {
				s.logger.Error("failed to delete docker container", zap.String("container_id", container.ContainerId), zap.Error(err))
			}
		}
	} else {
		result.SuccessCount += len(containers)
		for _, container := range containers {
			result.SuccessContainers = append(result.SuccessContainers, container.ContainerName)
		}
		s.logger.Info("containers imported successfully")
	}

	if err := s.redisClient.Set(ctx, "containers", redisContainers); err != nil {
		s.logger.Error("failed to update containers in redis", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *containerService) Export(ctx context.Context, filter dto.ContainerFilter, from int, to int, sort dto.ContainerSort) ([]byte, error) {
	if from < 1 {
		err := errors.New("invalid range")
		s.logger.Error("failed to export containers", zap.Error(err))
		return nil, err
	}
	limit := max(to-from+1, -1)

	containers, _, err := s.containerRepo.View(filter, from, limit, sort)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheetName := time.Now().Format(time.DateOnly)
	f.SetSheetName("Sheet1", sheetName)

	headers := []string{"Container ID", "Container Name", "Status", "IPv4", "Created At"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(sheetName, cell, h)
	}

	for idx, container := range containers {
		row := idx + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), container.ContainerId)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), container.ContainerName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), container.Status)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), container.Ipv4)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), container.CreatedAt.Format(time.RFC3339))
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		s.logger.Error("failed to export containers", zap.Error(err))
		return nil, err
	}
	s.logger.Info("containers exported successfully")
	return buf.Bytes(), nil
}
