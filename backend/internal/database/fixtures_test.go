package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type testStore struct{ Pool *pgxpool.Pool }

func newTestStore(ctx context.Context, dsn string) (*testStore, error) {
	p, err := Open(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err = Migrate(ctx, p); err != nil {
		p.Close()
		return nil, err
	}
	return &testStore{p}, nil
}
