package interfaces

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"github.com/vnFuhung2903/vcs-container-management-service/dto"
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/repositories"
)

type IKafkaConsumer interface {
	Consume(ctx context.Context) error
}

type kafkaConsumer struct {
	reader *kafka.Reader
	repo   repositories.IContainerRepository
}

func NewKafkaConsumer(reader *kafka.Reader, repo repositories.IContainerRepository) IKafkaConsumer {
	return &kafkaConsumer{reader: reader, repo: repo}
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
	return c.repo.Update(updateStatus.ContainerId, updateStatus.Status, updateStatus.Ipv4)
}
