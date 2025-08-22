package interfaces

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/vnFuhung2903/vcs-container-management-service/entities"
)

type IRedisClient interface {
	Set(ctx context.Context, key string, value []entities.ContainerWithStatus) error
	Get(ctx context.Context, key string) ([]entities.ContainerWithStatus, error)
	Del(ctx context.Context, key string) error
}

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(client *redis.Client) IRedisClient {
	return &RedisClient{client: client}
}

func (c *RedisClient) Set(ctx context.Context, key string, value []entities.ContainerWithStatus) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, bytes, 0).Err()
}

func (c *RedisClient) Get(ctx context.Context, key string) ([]entities.ContainerWithStatus, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return []entities.ContainerWithStatus{}, nil
	} else if err != nil {
		return nil, err
	}

	var result []entities.ContainerWithStatus
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *RedisClient) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
