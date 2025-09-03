package interfaces

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"github.com/vnFuhung2903/vcs-container-management-service/infrastructures/messages"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/env"
)

type KafkaConsumerSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	ctx      context.Context
	consumer IKafkaConsumer
}

func (suite *KafkaConsumerSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.ctx = context.Background()

	kafkaEnv := env.KafkaEnv{
		KafkaAddress: "localhost:9092",
	}
	factory := messages.NewKafkaFactory(kafkaEnv)
	reader, err := factory.ConnectKafkaReader("test-topic")
	suite.Require().NoError(err)

	suite.consumer = NewKafkaConsumer(reader)
}

func (suite *KafkaConsumerSuite) TestConsumeMessage() {
	_, err := suite.consumer.Consume(suite.ctx)
	suite.Error(err)
}

func TestKafkaConsumerSuite(t *testing.T) {
	suite.Run(t, new(KafkaConsumerSuite))
}
