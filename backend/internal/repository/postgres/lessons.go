// MIVA-DATA: persistence contracts and PostgreSQL implementations.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"english-app/internal/model"
	"english-app/internal/platform/identity"
	"english-app/internal/platform/jsonutil"
)

func (s *Lessons) Find(ctx context.Context, uid, id string) (*model.LessonRecord, error) {
	var r model.LessonRecord
	var raw, draft, attempt []byte
	var day time.Time
	where := "l.id=$2"
	param := id
	if id == "" {
		where = "$2::text=''"
	}
	err := s.db.QueryRow(
		ctx,
		`SELECT l.id,l.day,l.payload,l.draft,l.seconds,l.last_tick,a.payload FROM english_app.lessons l LEFT JOIN english_app.attempts a ON a.lesson_id=l.id AND a.user_id=l.user_id WHERE l.user_id=$1 AND `+where+` ORDER BY l.created_at DESC,l.id DESC LIMIT 1`,
		uid,
		param,
	).Scan(&r.ID, &day, &raw, &draft, &r.Seconds, &r.LastTick, &attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Day = day.Format("2006-01-02")
	if err = json.Unmarshal(raw, &r.Content); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(draft, &r.Draft); err != nil {
		return nil, err
	}
	if attempt != nil {
		r.Attempt = &model.Attempt{}
		if err = json.Unmarshal(attempt, r.Attempt); err != nil {
			return nil, err
		}
	}
	return &r, nil
}

func (s *Lessons) Create(ctx context.Context, uid, requestID string, l model.Lesson) (*model.LessonRecord, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(
		ctx,
		`INSERT INTO english_app.lessons(id,user_id,day,payload,request_id) VALUES($1,$2,$3,$4,$5) ON CONFLICT(user_id,request_id) DO UPDATE SET request_id=EXCLUDED.request_id RETURNING id`,
		identity.NewID(),
		uid,
		model.DayKey(time.Now()),
		jsonutil.Marshal(l),
		requestID,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	for _, w := range l.Vocabulary {
		w.ID = ""
		w.Source = "ai"
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO english_app.words VALUES($1,$2,$3,$4) ON CONFLICT(user_id,word) DO UPDATE SET payload=EXCLUDED.payload`,
			identity.NewID(),
			uid,
			strings.ToLower(w.Word),
			jsonutil.Marshal(w),
		); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.Find(ctx, uid, id)
}
