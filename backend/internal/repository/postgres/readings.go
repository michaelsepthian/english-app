// MIVA-DATA: persistence contracts and PostgreSQL implementations.
package postgres

import (
	"context"
	"encoding/json"

	"english-app/internal/model"
	"english-app/internal/platform/jsonutil"
)

func (r *Readings) List(ctx context.Context, uid, lesson string) ([]model.ReadingCheck, error) {
	rows, err := r.db.Query(ctx, "SELECT payload FROM english_app.reading_checks WHERE user_id=$1 AND lesson_id=$2 ORDER BY created_at DESC LIMIT 10", uid, lesson)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ReadingCheck{}
	for rows.Next() {
		var raw []byte
		var check model.ReadingCheck
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &check); err != nil {
			return nil, err
		}
		out = append(out, check)
	}
	return out, rows.Err()
}
func (r *Readings) Create(ctx context.Context, uid, lesson string, check model.ReadingCheck) error {
	_, err := r.db.Exec(ctx, "INSERT INTO english_app.reading_checks(id,user_id,lesson_id,source_id,payload) VALUES($1,$2,$3,$4,$5)", check.ID, uid, lesson, check.SourceID, jsonutil.Marshal(check))
	return err
}
