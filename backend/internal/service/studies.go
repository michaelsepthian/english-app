// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"context"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/repository"
)

type planGenerator interface {
	GeneratePlan(context.Context, model.Settings, model.StudyPlan, func(string, int) error) (model.Lesson, error)
}
type StudyService struct {
	jobs    repository.Jobs
	users   repository.Users
	lessons repository.Lessons
	tutor   *TutorService
	limits  *Limiter
	workers chan struct{}
}

func (s *StudyService) Job(ctx context.Context, uid, id string) (*model.StudyJob, *model.LessonRecord, error) {
	job, err := s.jobs.Find(ctx, uid, id)
	if err != nil {
		return nil, nil, err
	}
	var lesson *model.LessonRecord
	if job != nil && job.LessonID != nil && job.Status == "done" {
		lesson, err = s.lessons.Find(ctx, uid, *job.LessonID)
	}
	return job, lesson, err
}

// Prepare is idempotent per user and request. Accepted reports work still in progress.
func (s *StudyService) Prepare(ctx context.Context, uid string, in model.PrepareRequest) (job *model.StudyJob, accepted bool, err error) {
	if !idPattern.MatchString(in.RequestID) {
		return nil, false, apperror.NewError(400, "Sesi belum valid.")
	}
	if err = in.Plan.Validate(); err != nil {
		return nil, false, apperror.NewError(400, err.Error())
	}
	job, err = s.jobs.Find(ctx, uid, "")
	if err != nil {
		return nil, false, err
	}
	if job != nil && (job.Status == "queued" || job.Status == "running") {
		return job, true, nil
	}
	job, err = s.jobs.FindByRequest(ctx, uid, in.RequestID)
	if err != nil {
		return nil, false, err
	}
	if job != nil {
		return job, false, nil
	}
	if _, err = s.tutor.Client(); err != nil {
		return nil, false, err
	}
	if err = s.limits.Take(ctx, "study-prepare:"+uid, 8, time.Hour); err != nil {
		return nil, false, err
	}
	if err = s.jobs.Create(ctx, uid, in.RequestID, in.Plan); err != nil {
		return nil, false, err
	}
	job, err = s.jobs.FindByRequest(ctx, uid, in.RequestID)
	if err != nil {
		return nil, false, err
	}
	if job == nil {
		return nil, false, apperror.NewError(500, "Sesi belum dapat disiapkan.")
	}
	u, err := s.users.FindByID(ctx, uid)
	if err != nil {
		return nil, false, err
	}
	if job.Status == "queued" {
		go s.run(uid, job.RequestID, job.ID, u.Settings, job.Plan)
	}
	return job, true, nil
}
func (s *StudyService) run(uid, requestID, id string, settings model.Settings, plan model.StudyPlan) {
	fail := func(message string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.jobs.Fail(ctx, uid, id, message)
	}
	select {
	case s.workers <- struct{}{}:
		defer func() { <-s.workers }()
	default:
		fail("Sesi sedang penuh. Coba lagi sebentar.")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	claimed, err := s.jobs.Claim(ctx, uid, id)
	if err != nil || !claimed {
		return
	}
	ai, err := s.tutor.Client()
	if err != nil {
		fail("Materi belum dapat disiapkan. Coba kembali sebentar lagi.")
		return
	}
	generator, ok := ai.(planGenerator)
	if !ok {
		fail("Materi pilihan belum tersedia.")
		return
	}
	lesson, err := generator.GeneratePlan(ctx, settings, plan, func(stage string, progress int) error {
		if e := s.tutor.quota(ctx, uid); e != nil {
			return e
		}
		return s.jobs.Progress(ctx, uid, id, stage, progress)
	})
	if err != nil {
		fail("Materi belum lengkap. Coba siapkan sesi kembali; pilihanmu tetap tersimpan.")
		return
	}
	if err = lesson.Validate(); err != nil {
		fail("Materi belum sesuai pilihan. Coba siapkan kembali.")
		return
	}
	saved, err := s.lessons.Create(ctx, uid, requestID, lesson)
	if err != nil {
		fail("Materi belum berhasil disimpan. Coba kembali.")
		return
	}
	_ = s.jobs.Complete(ctx, uid, id, saved.ID)
}
