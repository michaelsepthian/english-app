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
	var raw, settings []byte
	values := append([]any{uid}, args...)
	err := r.db.QueryRow(ctx, "SELECT id,request_id,status,stage,progress,error,lesson_id,plan,settings,generator_version,attempt FROM english_app.study_jobs WHERE user_id=$1 "+where, values...).Scan(&job.ID, &job.RequestID, &job.Status, &job.Stage, &job.Progress, &job.Error, &job.LessonID, &raw, &settings, &job.GeneratorVersion, &job.Attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &job.Plan); err != nil {
		return nil, err
	}
	if settings != nil {
		job.Settings = &model.Settings{}
		if err = json.Unmarshal(settings, job.Settings); err != nil {
			return nil, err
		}
		job.Goal, job.Level = job.Settings.Goal, job.Settings.Level
	}
	job.CanResume = job.Status == "failed" && (job.GeneratorVersion == 0 || job.GeneratorVersion == model.GeneratorVersion)
	job.MaxAttempts = model.PreparationAttempts
	return &job, nil
}
func (r *Jobs) Find(ctx context.Context, uid, id string) (*model.StudyJob, error) {
	if id == "" {
		return r.find(ctx, uid, "ORDER BY (status IN ('queued','running')) DESC,created_at DESC,id DESC LIMIT 1")
	}
	return r.find(ctx, uid, "AND id=$2", id)
}
func (r *Jobs) FindByRequest(ctx context.Context, uid, id string) (*model.StudyJob, error) {
	return r.find(ctx, uid, "AND request_id=$2", id)
}
func (r *Jobs) Create(ctx context.Context, uid, request string, plan model.StudyPlan, settings model.Settings) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT id FROM english_app.users WHERE id=$1 FOR UPDATE", uid); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO english_app.study_jobs(id,user_id,request_id,plan,settings,generator_version) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING", identity.NewID(), uid, request, jsonutil.Marshal(plan), jsonutil.Marshal(settings), model.GeneratorVersion)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
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
	_, err := r.db.Exec(ctx, "UPDATE english_app.study_jobs SET status='failed',error='Persiapan terhenti saat aplikasi dimulai ulang. Materi tersimpan dapat dilanjutkan.' WHERE status IN ('queued','running')")
	return err
}

// Requeue serializes against both prepare and other resume requests for this user.
func (r *Jobs) Requeue(ctx context.Context, uid, id string, settings model.Settings) (bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT id FROM english_app.users WHERE id=$1 FOR UPDATE", uid); err != nil {
		return false, err
	}
	result, err := tx.Exec(ctx, `UPDATE english_app.study_jobs SET status='queued',error='',attempt=0,
 settings=COALESCE(settings,$3::jsonb),generator_version=$4,updated_at=now()
 WHERE user_id=$1 AND id=$2 AND status='failed' AND generator_version IN (0,$4)
 AND NOT EXISTS(SELECT 1 FROM english_app.study_jobs WHERE user_id=$1 AND status IN ('queued','running'))`, uid, id, jsonutil.Marshal(settings), model.GeneratorVersion)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() == 1, tx.Commit(ctx)
}
func (r *Jobs) Attempt(ctx context.Context, uid, id, stage string, progress, attempt int) error {
	_, err := r.db.Exec(ctx, `UPDATE english_app.study_jobs SET stage=$3,progress=GREATEST(progress,$4),attempt=$5,updated_at=now() WHERE user_id=$1 AND id=$2 AND status='running'`, uid, id, stage, progress, attempt)
	return err
}
func (r *Jobs) Checkpoints(ctx context.Context, uid, id string) (map[string]json.RawMessage, error) {
	rows, err := r.db.Query(ctx, `SELECT c.step_key,c.payload FROM english_app.study_job_checkpoints c JOIN english_app.study_jobs j ON j.id=c.job_id WHERE j.user_id=$1 AND j.id=$2 AND c.generator_version=$3`, uid, id, model.GeneratorVersion)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var key string
		var raw []byte
		if err = rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		out[key] = raw
	}
	return out, rows.Err()
}
func (r *Jobs) SaveCheckpoint(ctx context.Context, uid, id, key string, payload json.RawMessage, progress int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var owned string
	if err = tx.QueryRow(ctx, "SELECT id FROM english_app.study_jobs WHERE user_id=$1 AND id=$2 AND status='running' FOR UPDATE", uid, id).Scan(&owned); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO english_app.study_job_checkpoints(job_id,step_key,generator_version,payload) VALUES($1,$2,$3,$4) ON CONFLICT(job_id,step_key) DO NOTHING`, id, key, model.GeneratorVersion, []byte(payload)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE english_app.study_jobs SET progress=GREATEST(progress,$3),updated_at=now() WHERE user_id=$1 AND id=$2`, uid, id, progress); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
