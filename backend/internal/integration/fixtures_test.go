package integration

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"english-app/internal/database"
	"english-app/internal/handler"
	"english-app/internal/repository/postgres"
	"english-app/internal/service"
)

type testStore struct{ Pool *pgxpool.Pool }

func NewStore(ctx context.Context, dsn string) (*testStore, error) {
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err = database.Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return &testStore{pool}, nil
}
func newTestServer(store *testStore, ai service.AI, cfg service.Config) *handler.Server {
	return &handler.Server{Services: service.New(postgres.New(store.Pool), ai, nil, cfg), Config: handler.Config{Origin: "http://localhost:5180"}}
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
