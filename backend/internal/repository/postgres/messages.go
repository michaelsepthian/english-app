// MIVA-DATA: persistence contracts and PostgreSQL implementations.
package postgres

import (
	"context"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/identity"
)

func (s *Messages) List(ctx context.Context, uid string) ([]model.Message, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT id,role,content,created_at FROM(SELECT * FROM english_app.messages WHERE user_id=$1 ORDER BY seq DESC LIMIT 60)m ORDER BY seq`,
		uid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Message{}
	for rows.Next() {
		var m model.Message
		var t time.Time
		if err = rows.Scan(&m.ID, &m.Role, &m.Content, &t); err != nil {
			return nil, err
		}
		m.CreatedAt = t.Format(time.RFC3339)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Messages) CreateExchange(ctx context.Context, uid, text, reply string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, v := range []string{text, reply} {
		role := "user"
		if i == 1 {
			role = "assistant"
		}
		if _, err = tx.Exec(
			ctx,
			"INSERT INTO english_app.messages(id,user_id,role,content) VALUES($1,$2,$3,$4)",
			identity.NewID(),
			uid,
			role,
			v,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
