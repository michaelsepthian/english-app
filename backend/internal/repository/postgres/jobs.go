// MIVA-DATA: persistence contracts and PostgreSQL implementations.
package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"english-app/internal/model"
	"english-app/internal/platform/identity"
	"english-app/internal/platform/jsonutil"
)

func (r *Jobs) find(ctx context.Context, uid, where string, args ...any) (*model.StudyJob, error) {
	var job model.StudyJob
	var raw []byte
	values := append([]any{uid}, args...)
	err := r.db.QueryRow(ctx, "SELECT id,request_id,status,stage,progress,error,lesson_id,plan FROM english_app.study_jobs WHERE user_id=$1 "+where, values...).Scan(&job.ID, &job.RequestID, &job.Status, &job.Stage, &job.Progress, &job.Error, &job.LessonID, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &job.Plan); err != nil {
		return nil, err
	}
	return &job, nil
}
func (r *Jobs) Find(ctx context.Context, uid, id string) (*model.StudyJob, error) {
	if id == "" {
		return r.find(ctx, uid, "ORDER BY created_at DESC LIMIT 1")
	}
	return r.find(ctx, uid, "AND id=$2", id)
}
func (r *Jobs) FindByRequest(ctx context.Context, uid, id string) (*model.StudyJob, error) {
	return r.find(ctx, uid, "AND request_id=$2", id)
}
func (r *Jobs) Create(ctx context.Context, uid, request string, plan model.StudyPlan) error {
	_, err := r.db.Exec(ctx, "INSERT INTO english_app.study_jobs(id,user_id,request_id,plan) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", identity.NewID(), uid, request, jsonutil.Marshal(plan))
	return err
}
func (r *Jobs) Claim(ctx context.Context, uid, id string) (bool, error) {
	result, err := r.db.Exec(ctx, "UPDATE english_app.study_jobs SET status='running',updated_at=now() WHERE user_id=$1 AND id=$2 AND status='queued'", uid, id)
	return result.RowsAffected() == 1, err
}
func (r *Jobs) Progress(ctx context.Context, uid, id, stage string, progress int) error {
	_, err := r.db.Exec(ctx, "UPDATE english_app.study_jobs SET stage=$3,progress=$4,updated_at=now() WHERE user_id=$1 AND id=$2", uid, id, stage, progress)
	return err
}
func (r *Jobs) Fail(ctx context.Context, uid, id, message string) error {
	_, err := r.db.Exec(ctx, "UPDATE english_app.study_jobs SET status='failed',error=$3,updated_at=now() WHERE user_id=$1 AND id=$2 AND status IN ('queued','running')", uid, id, message)
	return err
}
func (r *Jobs) Complete(ctx context.Context, uid, id, lesson string) error {
	_, err := r.db.Exec(ctx, "UPDATE english_app.study_jobs SET status='done',stage='Sesi siap',progress=100,lesson_id=$3,updated_at=now() WHERE user_id=$1 AND id=$2", uid, id, lesson)
	return err
}
func (r *Jobs) RecoverInterrupted(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "UPDATE english_app.study_jobs SET status='failed',error='Persiapan terhenti saat aplikasi dimulai ulang. Silakan siapkan sesi kembali.' WHERE status IN ('queued','running')")
	return err
}
