// Package postgres implements the persistence ports. SQL belongs only here and in database migrations.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"english-app/internal/repository"
)

type Users struct{ db *pgxpool.Pool }
type Sessions struct{ db *pgxpool.Pool }
type Words struct{ db *pgxpool.Pool }
type Lessons struct{ db *pgxpool.Pool }
type Messages struct{ db *pgxpool.Pool }
type Jobs struct{ db *pgxpool.Pool }
type Readings struct{ db *pgxpool.Pool }
type Limits struct{ db *pgxpool.Pool }
type Health struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) repository.Repositories {
	return repository.Repositories{
		Users: &Users{db}, Sessions: &Sessions{db}, Words: &Words{db}, Lessons: &Lessons{db}, Messages: &Messages{db}, Jobs: &Jobs{db}, Readings: &Readings{db}, Limits: &Limits{db}, Health: &Health{db},
	}
}
func (r *Health) Ping(ctx context.Context) error { return r.db.Ping(ctx) }
func persistenceError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return repository.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return repository.ErrConflict
	}
	return err
}

var _ repository.Users = (*Users)(nil)
var _ repository.Sessions = (*Sessions)(nil)
var _ repository.Vocabulary = (*Words)(nil)
var _ repository.Lessons = (*Lessons)(nil)
var _ repository.Messages = (*Messages)(nil)
var _ repository.Jobs = (*Jobs)(nil)
var _ repository.Readings = (*Readings)(nil)
var _ repository.Limits = (*Limits)(nil)
