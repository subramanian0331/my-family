package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Client interface {
	Pool() *pgxpool.Pool
	Ping(ctx context.Context) error
	Close()
}

type client struct {
	pool *pgxpool.Pool
}

func New(databaseURL string) (Client, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}
	return &client{pool: pool}, nil
}

func (c *client) Pool() *pgxpool.Pool {
	return c.pool
}

func (c *client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *client) Close() {
	c.pool.Close()
}