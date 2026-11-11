// MIVA-MIGRATIONS: PostgreSQL connection and immutable migrations.
package database

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct{ Name, SQL, Checksum string }

func readMigrations(source fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(source, ".")
	if err != nil {
		return nil, err
	}
	out := []migration{}
	versions := map[string]bool{}
	pattern := regexp.MustCompile(`^[0-9]{4}_[a-z0-9_]+\.sql$`)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		name := entry.Name()
		if !pattern.MatchString(name) || versions[name[:4]] {
			return nil, fmt.Errorf("invalid or duplicate migration version: %s", name)
		}
		versions[name[:4]] = true
		raw, e := fs.ReadFile(source, name)
		if e != nil {
			return nil, e
		}
		if strings.TrimSpace(string(raw)) == "" {
			return nil, fmt.Errorf("empty migration: %s", name)
		}
		sum := sha256.Sum256(raw)
		out = append(out, migration{name, string(raw), fmt.Sprintf("%x", sum)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if len(out) == 0 {
		return nil, fmt.Errorf("no migrations embedded")
	}
	return out, nil
}

func embeddedMigrations() ([]migration, error) {
	source, err := fs.Sub(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}
	return readMigrations(source)
}

type migrationQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func appliedMigrations(ctx context.Context, q migrationQuerier) (map[string]string, error) {
	rows, err := q.Query(ctx, "SELECT name,checksum FROM english_app.schema_migrations ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("migration history unavailable; run migrate first: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, sum string
		if err = rows.Scan(&name, &sum); err != nil {
			return nil, err
		}
		out[name] = sum
	}
	return out, rows.Err()
}

func validateHistory(files []migration, applied map[string]string, complete bool) error {
	known := map[string]bool{}
	missing := false
	for _, file := range files {
		known[file.Name] = true
		sum, ok := applied[file.Name]
		if !ok {
			missing = true
			if complete {
				return fmt.Errorf("pending migration: %s", file.Name)
			}
			continue
		}
		if sum != file.Checksum {
			return fmt.Errorf("applied migration was modified: %s", file.Name)
		}
		if missing {
			return fmt.Errorf("migration history is out of order: %s", file.Name)
		}
	}
	for name := range applied {
		if !known[name] {
			return fmt.Errorf("database has a newer or unknown migration: %s", name)
		}
	}
	return nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := embeddedMigrations()
	if err != nil {
		return err
	}
	return applyMigrations(ctx, pool, files)
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool, files []migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SET LOCAL lock_timeout='15s'; SET LOCAL statement_timeout='120s'; SELECT pg_advisory_xact_lock(741926103)"); err != nil {
		return fmt.Errorf("migration lock unavailable: %w", err)
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='english_app')").Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = tx.Exec(ctx, "CREATE SCHEMA english_app"); err != nil {
			return fmt.Errorf("create english_app schema as the database administrator first: %w", err)
		}
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS english_app.schema_migrations(name text PRIMARY KEY,checksum text NOT NULL,applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("migration history setup failed: %w", err)
	}
	applied, err := appliedMigrations(ctx, tx)
	if err != nil {
		return err
	}
	if err = validateHistory(files, applied, false); err != nil {
		return err
	}
	for _, file := range files {
		if _, ok := applied[file.Name]; ok {
			continue
		}
		if _, err = tx.Exec(ctx, file.SQL); err != nil {
			return fmt.Errorf("migration %s failed: %w", file.Name, err)
		}
		if _, err = tx.Exec(
			ctx,
			"INSERT INTO english_app.schema_migrations(name,checksum) VALUES($1,$2)",
			file.Name,
			file.Checksum,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func CheckMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	files, err := embeddedMigrations()
	if err != nil {
		return err
	}
	applied, err := appliedMigrations(ctx, pool)
	if err != nil {
		return err
	}
	return validateHistory(files, applied, true)
}
