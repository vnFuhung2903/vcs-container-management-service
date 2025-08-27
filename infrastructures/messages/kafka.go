package messages

import (
	"github.com/segmentio/kafka-go"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/env"
)

type IKafkaFactory interface {
	ConnectKafkaReader(topic string) (*kafka.Reader, error)
}

type kafkaFactory struct {
	address string
}

func NewKafkaFactory(env env.KafkaEnv) IKafkaFactory {
	return &kafkaFactory{
		address: env.KafkaAddress,
	}
}

func (f *kafkaFactory) ConnectKafkaReader(topic string) (*kafka.Reader, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{f.address},
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return reader, nil
}
