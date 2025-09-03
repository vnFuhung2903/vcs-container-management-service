package workers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/suite"
	"github.com/vnFuhung2903/vcs-container-management-service/dto"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
	"github.com/vnFuhung2903/vcs-container-management-service/mocks/interfaces"
	"github.com/vnFuhung2903/vcs-container-management-service/mocks/logger"
	"github.com/vnFuhung2903/vcs-container-management-service/mocks/repositories"
)

type ConsumerWorkerSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	mockKafka  *interfaces.MockIKafkaConsumer
	mockRedis  *interfaces.MockIRedisClient
	mockRepo   *repositories.MockIContainerRepository
	mockLogger *logger.MockILogger
	worker     IConsumerWorker
}

func (s *ConsumerWorkerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockKafka = interfaces.NewMockIKafkaConsumer(s.ctrl)
	s.mockRedis = interfaces.NewMockIRedisClient(s.ctrl)
	s.mockRepo = repositories.NewMockIContainerRepository(s.ctrl)
	s.mockLogger = logger.NewMockILogger(s.ctrl)

	s.worker = NewConsumerWorker(
		s.mockKafka,
		s.mockRedis,
		s.mockRepo,
		s.mockLogger,
	)
}

func (s *ConsumerWorkerSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *ConsumerWorkerSuite) TestConsumerWorker() {
	update := dto.KafkaStatusUpdate{
		ContainerId: "abc123",
		Status:      entities.ContainerOn,
		Ipv4:        "1.2.3.4",
	}
	msgBytes, _ := json.Marshal(update)
	kafkaMsg := kafka.Message{Value: msgBytes}

	containers := []entities.ContainerWithStatus{
		{ContainerId: "abc123", Status: entities.ContainerOff},
	}

	s.mockKafka.EXPECT().
		Consume(gomock.Any()).
		Return(kafkaMsg, nil).
		AnyTimes()

	s.mockKafka.EXPECT().
		Consume(gomock.Any()).
		Return(kafka.Message{}, context.Canceled).
		AnyTimes()

	s.mockRedis.EXPECT().
		Get(gomock.Any(), "containers").
		Return(containers, nil).
		AnyTimes()

	s.mockRedis.EXPECT().
		Set(gomock.Any(), "containers", gomock.Any()).
		Return(nil).
		AnyTimes()

	s.mockRepo.EXPECT().
		Update(update.ContainerId, update.Status, update.Ipv4).
		Return(nil).
		AnyTimes()

	s.mockLogger.EXPECT().
		Info("consumed message from kafka successfully", gomock.Any(), gomock.Any()).
		AnyTimes()

	s.mockLogger.EXPECT().
		Info("consumer worker stopped").
		AnyTimes()

	s.worker.Start()
	time.Sleep(50 * time.Millisecond)
	s.worker.Stop()
}

func (s *ConsumerWorkerSuite) TestConsumeKafkaError() {
	s.mockKafka.EXPECT().
		Consume(gomock.Any()).
		Return(kafka.Message{}, errors.New("kafka error")).
		AnyTimes()

	s.mockLogger.EXPECT().
		Error("failed to consume message", gomock.Any()).
		AnyTimes()

	s.mockLogger.EXPECT().
		Info("consumer worker stopped").
		AnyTimes()

	s.worker.Start()
	time.Sleep(10 * time.Millisecond)
	s.worker.Stop()
}

func (s *ConsumerWorkerSuite) TestConsumeInvalidJSON() {
	s.mockKafka.EXPECT().
		Consume(gomock.Any()).
		Return(kafka.Message{}, nil).
		AnyTimes()

	s.mockLogger.EXPECT().
		Error("failed to unmarshal message", gomock.Any()).
		AnyTimes()

	s.mockLogger.EXPECT().
		Info("consumer worker stopped").
		AnyTimes()

	s.worker.Start()
	time.Sleep(10 * time.Millisecond)
	s.worker.Stop()
}

func (s *ConsumerWorkerSuite) TestConsumerRedisGetError() {
	update := dto.KafkaStatusUpdate{
		ContainerId: "abc123",
		Status:      entities.ContainerOn,
		Ipv4:        "1.2.3.4",
	}
	msgBytes, _ := json.Marshal(update)
	kafkaMsg := kafka.Message{Value: msgBytes}

	s.mockKafka.EXPECT().
		Consume(gomock.Any()).
		Return(kafkaMsg, nil).
		AnyTimes()

	s.mockRedis.EXPECT().
		Get(gomock.Any(), "containers").
		Return(nil, errors.New("redis error")).
		AnyTimes()

	s.mockLogger.EXPECT().
		Error("failed to retrieve containers from redis", gomock.Any()).
		AnyTimes()

	s.mockLogger.EXPECT().
		Info("consumer worker stopped").
		AnyTimes()

	s.worker.Start()
	time.Sleep(50 * time.Millisecond)
	s.worker.Stop()
}

func (s *ConsumerWorkerSuite) TestConsumerRedisSetError() {
	update := dto.KafkaStatusUpdate{
		ContainerId: "abc123",
		Status:      entities.ContainerOn,
		Ipv4:        "1.2.3.4",
	}
	msgBytes, _ := json.Marshal(update)
	kafkaMsg := kafka.Message{Value: msgBytes}

	s.mockKafka.EXPECT().
		Consume(gomock.Any()).
		Return(kafkaMsg, nil).
		AnyTimes()

	s.mockRedis.EXPECT().
		Get(gomock.Any(), "containers").
		Return([]entities.ContainerWithStatus{}, nil).
		AnyTimes()

	s.mockRedis.EXPECT().
		Set(gomock.Any(), "containers", gomock.Any()).
		Return(errors.New("redis error")).
		AnyTimes()

	s.mockLogger.EXPECT().
		Error("failed to update containers in redis", gomock.Any()).
		AnyTimes()

	s.mockLogger.EXPECT().
		Info("consumer worker stopped").
		AnyTimes()

	s.worker.Start()
	time.Sleep(50 * time.Millisecond)
	s.worker.Stop()
}

func (s *ConsumerWorkerSuite) TestConsumerRepoError() {
	update := dto.KafkaStatusUpdate{
		ContainerId: "abc123",
		Status:      entities.ContainerOn,
		Ipv4:        "1.2.3.4",
	}
	msgBytes, _ := json.Marshal(update)
	kafkaMsg := kafka.Message{Value: msgBytes}

	containers := []entities.ContainerWithStatus{
		{ContainerId: "abc123", Status: entities.ContainerOff},
	}

	s.mockKafka.EXPECT().
		Consume(gomock.Any()).
		Return(kafkaMsg, nil).
		AnyTimes()

	s.mockKafka.EXPECT().
		Consume(gomock.Any()).
		Return(kafka.Message{}, context.Canceled).
		AnyTimes()

	s.mockRedis.EXPECT().
		Get(gomock.Any(), "containers").
		Return(containers, nil).
		AnyTimes()

	s.mockRedis.EXPECT().
		Set(gomock.Any(), "containers", gomock.Any()).
		Return(nil).
		AnyTimes()

	s.mockRepo.EXPECT().
		Update(update.ContainerId, update.Status, update.Ipv4).
		Return(errors.New("repo error")).
		AnyTimes()

	s.mockLogger.EXPECT().
		Error("failed to update container", gomock.Any()).
		AnyTimes()

	s.mockLogger.EXPECT().
		Info("consumer worker stopped").
		AnyTimes()

	s.worker.Start()
	time.Sleep(50 * time.Millisecond)
	s.worker.Stop()
}

func TestConsumerWorkerSuite(t *testing.T) {
	suite.Run(t, new(ConsumerWorkerSuite))
}
