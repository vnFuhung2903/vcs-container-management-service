package interfaces

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"github.com/vnFuhung2903/vcs-container-management-service/dto"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/logger"
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/repositories"
	"go.uber.org/zap"
)

type IKafkaConsumer interface {
	Consume(ctx context.Context) error
}

type kafkaConsumer struct {
	reader      *kafka.Reader
	redisClient IRedisClient
	repo        repositories.IContainerRepository
	logger      logger.ILogger
}

func NewKafkaConsumer(reader *kafka.Reader, redisClient IRedisClient, repo repositories.IContainerRepository, logger logger.ILogger) IKafkaConsumer {
	return &kafkaConsumer{reader: reader, redisClient: redisClient, repo: repo, logger: logger}
}

func (c *kafkaConsumer) Consume(ctx context.Context) error {
	message, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return err
	}
	updateStatus := dto.KafkaStatusUpdate{}
	if err := json.Unmarshal(message.Value, &updateStatus); err != nil {
		return err
	}
	c.logger.Info("Consumed message from Kafka", zap.String("container_id", updateStatus.ContainerId), zap.String("status", string(updateStatus.Status)), zap.String("ipv4", updateStatus.Ipv4))

	containers, err := c.redisClient.Get(ctx, "containers")
	if err != nil {
		c.logger.Error("failed to retrieve containers from redis", zap.Error(err))
		return err
	}

	for i := range containers {
		if containers[i].ContainerId == updateStatus.ContainerId {
			containers[i].Status = updateStatus.Status
		}
	}

	if err := c.redisClient.Set(ctx, "containers", containers); err != nil {
		c.logger.Error("failed to set containers to redis", zap.Error(err))
		return err
	}
	return c.repo.Update(updateStatus.ContainerId, updateStatus.Status, updateStatus.Ipv4)
}
