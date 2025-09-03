package messages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/env"
)

type MessagesSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *MessagesSuite) SetupSuite() {
	suite.ctx = context.Background()
}

func TestMessagesSuite(t *testing.T) {
	suite.Run(t, new(MessagesSuite))
}

func (suite *MessagesSuite) TestConnectKafkaReader() {
	kafkaEnv := env.KafkaEnv{
		KafkaAddress: "localhost:9092",
	}
	factory := NewKafkaFactory(kafkaEnv)
	reader, err := factory.ConnectKafkaReader("test-topic")
	suite.NoError(err)
	suite.NotNil(reader)
}
