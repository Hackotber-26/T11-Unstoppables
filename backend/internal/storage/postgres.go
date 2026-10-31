package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	Pool *pgxpool.Pool
}

func NewPostgresStore(dsn string) (*PostgresStore, error) {
	if dsn == "" {
		return &PostgresStore{}, nil
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	if err = pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresStore{Pool: pool}, nil
}

func (s *PostgresStore) Close() error {
	if s != nil && s.Pool != nil {
		s.Pool.Close()
	}
	return nil
}
