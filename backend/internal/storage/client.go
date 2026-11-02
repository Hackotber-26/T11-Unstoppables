package storage

import (
	"fmt"
	"t11-unstoppables/internal/config"
)

type Client struct {
	Postgres *PostgresStore
	Redis    *RedisStore
}

func NewClient(cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	pg, err := NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	cache, err := NewRedisStore(cfg.RedisURL)
	if err != nil {
		pg.Close()
		return nil, err
	}
	return &Client{Postgres: pg, Redis: cache}, nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	if c.Postgres != nil {
		if err := c.Postgres.Close(); err != nil {
			return err
		}
	}
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			return err
		}
	}
	return nil
}
