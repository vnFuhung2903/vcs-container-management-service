package interfaces

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type IRedisClient interface {
	Set(ctx context.Context, key string, value []string) error
	Get(ctx context.Context, key string) ([]string, error)
	Del(ctx context.Context, key string) error
}

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(client *redis.Client) IRedisClient {
	return &RedisClient{client: client}
}

func (c *RedisClient) Set(ctx context.Context, key string, value []string) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, bytes, 0).Err()
}

func (c *RedisClient) Get(ctx context.Context, key string) ([]string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return []string{}, nil
	} else if err != nil {
		return nil, err
	}

	var result []string
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *RedisClient) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
