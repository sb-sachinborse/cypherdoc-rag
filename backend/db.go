package main

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

// connectDB retries for a while so the backend survives the database
// finishing its first-boot initialisation.
func connectDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 10
	var lastErr error
	for i := 0; i < 30; i++ {
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				if _, err = pool.Exec(ctx, schemaSQL); err != nil {
					pool.Close()
					return nil, fmt.Errorf("migrate: %w", err)
				}
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("database not reachable: %w", lastErr)
}
