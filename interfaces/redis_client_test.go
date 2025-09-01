package interfaces

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
)

func TestRedisClient(t *testing.T) {
	opt := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15,
	}
	rds := redis.NewClient(opt)
	assert.NotNil(t, rds)

	redisClient := NewRedisClient(rds)
	ctx := context.Background()
	testKey := "test-key"

	testData := []entities.ContainerWithStatus{
		{
			ContainerId: "container-1",
			Status:      entities.ContainerOn,
		},
		{
			ContainerId: "container-2",
			Status:      entities.ContainerOff,
		},
	}

	_ = redisClient.Del(ctx, testKey)

	err := redisClient.Set(ctx, testKey, testData)
	assert.NoError(t, err)

	result, err := redisClient.Get(ctx, testKey)
	assert.NoError(t, err)
	assert.Equal(t, testData, result)

	err = redisClient.Del(ctx, testKey)
	assert.NoError(t, err)

	result, err = redisClient.Get(ctx, testKey)
	assert.NoError(t, err)
	assert.Empty(t, result)
}
