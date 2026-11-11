// MIVA-DATA: persistence contracts and PostgreSQL implementations.
package postgres

import (
	"context"
	"encoding/json"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/jsonutil"
)

func (s *Lessons) SaveDraft(ctx context.Context, uid, id string, answers map[string]int) error {
	_, err := s.db.Exec(ctx, `UPDATE english_app.lessons SET draft=$1 WHERE id=$2 AND user_id=$3 AND NOT EXISTS(SELECT 1 FROM english_app.attempts WHERE lesson_id=$2)`, jsonutil.Marshal(answers), id, uid)
	return err
}
func (s *Lessons) Heartbeat(ctx context.Context, uid, id string, active bool) (bool, error) {
	var tick any
	if active {
		tick = time.Now()
	}
	r, err := s.db.Exec(ctx, `UPDATE english_app.lessons SET seconds=LEAST(seconds+CASE WHEN last_tick IS NULL THEN 0 ELSE GREATEST(0,LEAST(20,FLOOR(EXTRACT(EPOCH FROM(now()-last_tick)))::int)) END,7200),last_tick=$1 WHERE id=$2 AND user_id=$3 AND NOT EXISTS(SELECT 1 FROM english_app.attempts WHERE lesson_id=$2)`, tick, id, uid)
	return r.RowsAffected() > 0, err
}
func (s *Lessons) CreateAttempt(ctx context.Context, uid, id string, a model.Attempt) error {
	_, err := s.db.Exec(ctx, `INSERT INTO english_app.attempts VALUES($1,$2,$3,$4) ON CONFLICT(lesson_id) DO NOTHING`, id, uid, model.DayKey(time.Now()), jsonutil.Marshal(a))
	return err
}
func (s *Lessons) SaveReview(ctx context.Context, uid, id string, a model.Attempt) error {
	_, err := s.db.Exec(ctx, `UPDATE english_app.attempts SET payload=$1 WHERE lesson_id=$2 AND user_id=$3`, jsonutil.Marshal(a), id, uid)
	return err
}
func (s *Lessons) RecentThemes(ctx context.Context, uid string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT payload->>'theme' FROM english_app.lessons WHERE user_id=$1 ORDER BY day DESC LIMIT 7`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var v string
		if err = rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Lessons) History(ctx context.Context, uid string) ([]model.LessonHistoryRecord, error) {
	rows, err := s.db.Query(ctx, `SELECT l.id,l.day::text,l.payload->>'title',jsonb_array_length(l.payload->'questions'),a.payload FROM english_app.lessons l LEFT JOIN english_app.attempts a ON a.lesson_id=l.id AND a.user_id=l.user_id WHERE l.user_id=$1 ORDER BY l.created_at DESC,l.id DESC LIMIT 50`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.LessonHistoryRecord{}
	for rows.Next() {
		var v model.LessonHistoryRecord
		var raw []byte
		if err = rows.Scan(&v.ID, &v.Day, &v.Title, &v.Total, &raw); err != nil {
			return nil, err
		}
		if raw != nil {
			v.Attempt = &model.Attempt{}
			if err = json.Unmarshal(raw, v.Attempt); err != nil {
				return nil, err
			}
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Lessons) Attempts(ctx context.Context, uid, start string) ([]model.AttemptRecord, error) {
	rows, err := s.db.Query(ctx, `SELECT day::text,payload FROM english_app.attempts WHERE user_id=$1 AND day>=$2 ORDER BY day`, uid, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.AttemptRecord{}
	for rows.Next() {
		var v model.AttemptRecord
		var raw []byte
		if err = rows.Scan(&v.Day, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &v.Attempt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
