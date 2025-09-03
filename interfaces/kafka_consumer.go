package interfaces

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type IKafkaConsumer interface {
	Consume(ctx context.Context) (kafka.Message, error)
}

type kafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(reader *kafka.Reader) IKafkaConsumer {
	return &kafkaConsumer{reader: reader}
}

func (c *kafkaConsumer) Consume(ctx context.Context) (kafka.Message, error) {
	return c.reader.ReadMessage(ctx)
}
