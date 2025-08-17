package docker

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
)

type DockerClientSuite struct {
	suite.Suite
	client IDockerClient
	ctx    context.Context
}

func (suite *DockerClientSuite) SetupTest() {
	suite.ctx = context.Background()

	client, err := NewDockerClient()
	suite.client = client
	suite.NoError(err)
}

func TestDockerClientSuite(t *testing.T) {
	suite.Run(t, new(DockerClientSuite))
}

func (suite *DockerClientSuite) TestContainerOnLifeCycle() {
	con, err := suite.client.Create(suite.ctx, "test-container", "nginx:stable-alpine-perl")
	suite.NoError(err)

	err = suite.client.Start(suite.ctx, con.ID)
	suite.NoError(err)

	status := suite.client.GetStatus(suite.ctx, con.ID)
	suite.Equal(entities.ContainerOn, status)
	ipv4 := suite.client.GetIpv4(suite.ctx, con.ID)
	suite.NotEqual("", ipv4)

	err = suite.client.Stop(suite.ctx, con.ID)
	suite.NoError(err)

	err = suite.client.Delete(suite.ctx, con.ID)
	suite.NoError(err)
}

func (suite *DockerClientSuite) TestContainerOffLifeCycle() {
	con, err := suite.client.Create(suite.ctx, "test-container", "nginx:stable-alpine-perl")
	suite.NoError(err)

	status := suite.client.GetStatus(suite.ctx, con.ID)
	suite.Equal(entities.ContainerOff, status)
	ipv4 := suite.client.GetIpv4(suite.ctx, con.ID)
	suite.Equal("", ipv4)

	err = suite.client.Delete(suite.ctx, con.ID)
	suite.NoError(err)
}

func (suite *DockerClientSuite) TestPullImageInvalidImage() {
	dockerClient := suite.client.(*DockerClient)
	err := dockerClient.PullImage(suite.ctx, "invalid/non-existent-image:invalid-tag")
	suite.Error(err)
	suite.Contains(strings.ToLower(err.Error()), "failed to pull image")
}

func (suite *DockerClientSuite) TestCreateContainerInvalidImage() {
	_, err := suite.client.Create(suite.ctx, "test-container", "invalid/non-existent-image")
	suite.Error(err)
}

func (suite *DockerClientSuite) TestGetStatusNonExistentContainer() {
	status := suite.client.GetStatus(suite.ctx, "non-existent-container-id")
	suite.Equal(entities.ContainerOff, status)
}

func (suite *DockerClientSuite) TestGetIpv4NonExistentContainer() {
	ipv4 := suite.client.GetIpv4(suite.ctx, "non-existent-container-id")
	suite.Equal("", ipv4)
}

func (suite *DockerClientSuite) TestStartNonExistentContainer() {
	err := suite.client.Start(suite.ctx, "non-existent-container-id")
	suite.Error(err)
}

func (suite *DockerClientSuite) TestStopNonExistentContainer() {
	err := suite.client.Stop(suite.ctx, "non-existent-container-id")
	suite.Error(err)
}

func (suite *DockerClientSuite) TestDeleteNonExistentContainer() {
	err := suite.client.Delete(suite.ctx, "non-existent-container-id")
	suite.T().Logf("Delete non-existent container result: %v", err)
}
