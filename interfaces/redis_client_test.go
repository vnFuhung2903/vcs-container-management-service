package interfaces

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisClient(t *testing.T) {
	opt := &redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}
	rds := redis.NewClient(opt)
	assert.NotNil(t, rds)

	redisClient := NewRedisClient(rds)

	err := redisClient.Set(context.Background(), "test-key", []string{"test-value"})
	assert.Error(t, err)

	_, err = redisClient.Get(context.Background(), "test-key")
	assert.Error(t, err)

	err = redisClient.Del(context.Background(), "test-key")
	assert.Error(t, err)
}
