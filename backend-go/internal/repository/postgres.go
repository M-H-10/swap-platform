// Package repository holds all the code that talks to PostgreSQL.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens a pool of connections to PostgreSQL and checks that the
// database answers. The pool is shared by the whole server.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing DATABASE_URL: %w", err)
	}

	cfg.MaxConns = 10                     // at most 10 open connections
	cfg.MaxConnIdleTime = 5 * time.Minute // close connections nobody uses

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// The pool connects lazily, so ping once now to fail fast if the
	// database is down or the password is wrong.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to PostgreSQL: %w", err)
	}

	return pool, nil
}
