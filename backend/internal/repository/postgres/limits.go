// MIVA-DATA: persistence contracts and PostgreSQL implementations.
package postgres

import (
	"context"
	"time"
)

func (r *Limits) Increment(ctx context.Context, key string, expires time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `INSERT INTO english_app.limits VALUES($1,1,$2) ON CONFLICT(key) DO UPDATE SET count=CASE WHEN english_app.limits.expires<now() THEN 1 ELSE english_app.limits.count+1 END,expires=CASE WHEN english_app.limits.expires<now() THEN EXCLUDED.expires ELSE english_app.limits.expires END RETURNING count`, key, expires).Scan(&count)
	return count, err
}
func (r *Limits) Acquire(ctx context.Context, key string, expires time.Time) (bool, error) {
	result, err := r.db.Exec(ctx, `INSERT INTO english_app.locks VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET expires=EXCLUDED.expires WHERE english_app.locks.expires<now()`, key, expires)
	return result.RowsAffected() == 1, err
}
func (r *Limits) Release(ctx context.Context, key string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM english_app.locks WHERE key=$1", key)
	return err
}
