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
	reader *kafka.Reader
	repo   repositories.IContainerRepository
	logger logger.ILogger
}

func NewKafkaConsumer(reader *kafka.Reader, repo repositories.IContainerRepository, logger logger.ILogger) IKafkaConsumer {
	return &kafkaConsumer{reader: reader, repo: repo, logger: logger}
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
	return c.repo.Update(updateStatus.ContainerId, updateStatus.Status, updateStatus.Ipv4)
}
