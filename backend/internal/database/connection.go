// MIVA-MIGRATIONS: PostgreSQL connection and immutable migrations.
package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("konfigurasi PostgreSQL tidak valid")
	}
	cfg.MaxConns = 3
	cfg.MinConns = 0
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.ConnConfig.RuntimeParams["search_path"] = "english_app,pg_catalog"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
