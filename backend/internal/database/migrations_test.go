package database

import (
	"context"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

func TestMigrationOrderingAndImmutableHistory(t *testing.T) {
	files, err := readMigrations(fstest.MapFS{"0002_next.sql": {Data: []byte("SELECT 2;")}, "0001_first.sql": {Data: []byte("SELECT 1;")}})
	if err != nil || files[0].Name != "0001_first.sql" {
		t.Fatalf("order: %+v %v", files, err)
	}
	if validateHistory(files, map[string]string{files[0].Name: files[0].Checksum}, false) != nil {
		t.Fatal("pending valid upgrade rejected")
	}
	for _, history := range []map[string]string{{files[0].Name: "tampered"}, {"9999_future.sql": "unknown"}, {files[1].Name: files[1].Checksum}} {
		if validateHistory(files, history, false) == nil {
			t.Fatal("invalid history accepted")
		}
	}
	if validateHistory(files, map[string]string{}, true) == nil {
		t.Fatal("server started with pending migrations")
	}
	if _, err = readMigrations(fstest.MapFS{"0001_a.sql": {Data: []byte("SELECT 1")}, "0001_b.sql": {Data: []byte("SELECT 2")}}); err == nil {
		t.Fatal("duplicate version accepted")
	}
}
func TestPostgresMigrationReplayLockAndRollback(t *testing.T) {
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("opt-in PostgreSQL tests")
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")),
		Host:   os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"),
		Path:   os.Getenv("DB_NAME"),
	}
	u.RawQuery = "sslmode=disable"
	store, err := newTestStore(t.Context(), u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Pool.Close()
	var before string
	if err = store.Pool.QueryRow(t.Context(), "SELECT string_agg(name||checksum||applied_at::text,',' ORDER BY name) FROM english_app.schema_migrations").Scan(&before); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- Migrate(t.Context(), store.Pool) }()
	}
	wg.Wait()
	close(errs)
	for err = range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var after string
	_ = store.Pool.QueryRow(t.Context(), "SELECT string_agg(name||checksum||applied_at::text,',' ORDER BY name) FROM english_app.schema_migrations").Scan(&after)
	if before != after {
		t.Fatal("replay changed applied migrations")
	}
	files, _ := embeddedMigrations()
	files = append(files, migration{
		Name:     "9999_transaction_test.sql",
		SQL:      "CREATE TABLE english_app.migration_rollback_test(id integer); SELECT 1/0;",
		Checksum: "test",
	})
	if err = applyMigrations(t.Context(), store.Pool, files); err == nil || !strings.Contains(err.Error(), "9999_transaction_test") {
		t.Fatal("failed migration not rejected", err)
	}
	var exists bool
	err = store.Pool.QueryRow(context.Background(), "SELECT to_regclass('english_app.migration_rollback_test') IS NOT NULL OR EXISTS(SELECT 1 FROM english_app.schema_migrations WHERE name='9999_transaction_test.sql')").Scan(&exists)
	if err != nil || exists {
		t.Fatalf("DDL/history were not rolled back: %v", err)
	}
	if err = CheckMigrations(t.Context(), store.Pool); err != nil {
		t.Fatal(err)
	}
}
