// MIVA-DATA: persistence contracts and PostgreSQL implementations.
package postgres

import (
	"context"
	"encoding/json"
	"strings"

	"english-app/internal/model"
	"english-app/internal/platform/identity"
	"english-app/internal/platform/jsonutil"
)

func (s *Words) List(ctx context.Context, uid string) ([]model.Word, error) {
	rows, err := s.db.Query(ctx, "SELECT id,payload FROM english_app.words WHERE user_id=$1 ORDER BY word", uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Word{}
	for rows.Next() {
		var w model.Word
		var raw []byte
		if err = rows.Scan(&w.ID, &raw); err != nil {
			return nil, err
		}
		id := w.ID
		if err = json.Unmarshal(raw, &w); err != nil {
			return nil, err
		}
		w.ID = id
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Words) Upsert(ctx context.Context, uid string, w model.Word) error {
	w.ID = ""
	w.Source = "ai"
	_, err := s.db.Exec(
		ctx,
		`INSERT INTO english_app.words VALUES($1,$2,$3,$4) ON CONFLICT(user_id,word) DO UPDATE SET payload=EXCLUDED.payload`,
		identity.NewID(),
		uid,
		strings.ToLower(w.Word),
		jsonutil.Marshal(w),
	)
	return err
}
