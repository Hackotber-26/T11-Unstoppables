package storage

import (
	"context"
	"fmt"

	redis "github.com/redis/go-redis/v9"
)

type RedisStore struct {
	Client *redis.Client
}

func NewRedisStore(rawURL string) (*RedisStore, error) {
	if rawURL == "" {
		return &RedisStore{}, nil
	}
	options, err := parseRedisOptions(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis options: %w", err)
	}
	client := redis.NewClient(options)
	if err := client.Ping(context.Background()).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisStore{Client: client}, nil
}

func parseRedisOptions(rawURL string) (*redis.Options, error) {
	return redis.ParseURL(rawURL)
}

func (s *RedisStore) Close() error {
	if s != nil && s.Client != nil {
		return s.Client.Close()
	}
	return nil
}
