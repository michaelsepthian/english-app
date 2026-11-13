// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/repository"
)

type planGenerator interface {
	GeneratePlanCheckpointed(context.Context, model.Settings, model.StudyPlan, model.PreparationHooks) (model.Lesson, error)
}
type StudyService struct {
	jobs          repository.Jobs
	users         repository.Users
	lessons       repository.Lessons
	tutor         *TutorService
	limits        *Limiter
	workers       chan struct{}
	workerContext context.Context
	cancelWorkers context.CancelFunc
	workerMu      sync.Mutex
	closing       bool
	workerWG      sync.WaitGroup
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
	u, err := s.users.FindByID(ctx, uid)
	if err != nil {
		return nil, false, err
	}
	if err = s.jobs.Create(ctx, uid, in.RequestID, in.Plan, u.Settings); err != nil {
		return nil, false, err
	}
	job, err = s.jobs.Find(ctx, uid, "")
	if err != nil {
		return nil, false, err
	}
	if job == nil {
		return nil, false, apperror.NewError(500, "Sesi belum dapat disiapkan.")
	}
	if job.Status == "queued" {
		s.launch(uid, job)
	}
	return job, true, nil
}

func (s *StudyService) Resume(ctx context.Context, uid, id string) (*model.StudyJob, error) {
	job, err := s.jobs.Find(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	if id == "" || job == nil {
		return nil, apperror.NewError(404, "Persiapan tidak ditemukan.")
	}
	if job.Status == "queued" || job.Status == "running" {
		return job, nil
	}
	if !job.CanResume {
		return nil, apperror.NewError(409, "Persiapan ini tidak dapat dilanjutkan. Buat rencana baru.")
	}
	if _, err = s.tutor.Client(); err != nil {
		return nil, err
	}
	settings := job.Settings
	if settings == nil {
		u, e := s.users.FindByID(ctx, uid)
		if e != nil {
			return nil, e
		}
		settings = &u.Settings
	}
	queued, err := s.jobs.Requeue(ctx, uid, id, *settings)
	if err != nil {
		return nil, err
	}
	if !queued {
		active, e := s.jobs.Find(ctx, uid, "")
		if e != nil {
			return nil, e
		}
		if active != nil && (active.Status == "queued" || active.Status == "running") {
			return active, nil
		}
		return nil, apperror.NewError(409, "Persiapan belum dapat dilanjutkan.")
	}
	job, err = s.jobs.Find(ctx, uid, id)
	if err == nil && job != nil {
		s.launch(uid, job)
	}
	return job, err
}
func (s *StudyService) launch(uid string, job *model.StudyJob) {
	s.workerMu.Lock()
	defer s.workerMu.Unlock()
	if s.closing {
		return
	}
	s.workerWG.Add(1)
	go func() { defer s.workerWG.Done(); s.run(uid, job) }()
}
func (s *StudyService) Shutdown(ctx context.Context) error {
	s.workerMu.Lock()
	s.closing = true
	if s.cancelWorkers != nil {
		s.cancelWorkers()
	}
	s.workerMu.Unlock()
	done := make(chan struct{})
	go func() { s.workerWG.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *StudyService) run(uid string, job *model.StudyJob) {
	id := job.ID
	fail := func(message, category string) {
		slog.Warn("preparation failed", "job", id, "category", category)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.jobs.Fail(ctx, uid, id, message); err != nil {
			slog.Error("preparation failure persistence", "job", id)
		}
	}
	ctx := s.workerContext
	if ctx == nil {
		ctx = context.Background()
	}
	claimed, err := s.jobs.Claim(ctx, uid, id)
	if err != nil {
		fail("Persiapan belum dapat dimulai. Coba lanjutkan kembali.", "storage")
		return
	}
	if !claimed {
		return
	}
	select {
	case s.workers <- struct{}{}:
		defer func() { <-s.workers }()
	default:
		fail("Sesi sedang penuh. Coba lagi sebentar.", "capacity")
		return
	}
	ai, err := s.tutor.Client()
	if err != nil {
		fail("Materi belum dapat disiapkan. Coba kembali sebentar lagi.", "configuration")
		return
	}
	generator, ok := ai.(planGenerator)
	if !ok || job.Settings == nil {
		fail("Materi pilihan belum tersedia. Buat rencana baru.", "configuration")
		return
	}
	checkpoints, err := s.jobs.Checkpoints(ctx, uid, id)
	if err != nil {
		fail("Progres belum dapat dimuat. Coba lanjutkan kembali.", "storage")
		return
	}
	var lesson model.Lesson
	if raw := checkpoints["final-lesson"]; raw != nil {
		err = json.Unmarshal(raw, &lesson)
	} else {
		lesson, err = generator.GeneratePlanCheckpointed(ctx, *job.Settings, job.Plan, model.PreparationHooks{
			JobID: id, Checkpoints: checkpoints,
			BeforeAttempt: func(key, stage string, progress, attempt int) error {
				if e := s.jobs.Attempt(ctx, uid, id, stage, progress, attempt); e != nil {
					return e
				}
				return s.tutor.quota(ctx, uid)
			},
			Save: func(key string, raw json.RawMessage, progress int) error {
				if e := s.jobs.SaveCheckpoint(ctx, uid, id, key, raw, progress); e != nil {
					slog.Warn("checkpoint failed", "job", id, "part", key, "category", "storage")
					return e
				}
				return nil
			},
		})
	}
	if err != nil {
		category := "generation"
		message := "Materi belum lengkap. Materi sebelumnya sudah tersimpan. Kamu dapat melanjutkan persiapan."
		if apperror.IsTimeout(err) {
			category = "timeout"
			message = "Bagian ini masih mengalami timeout. Materi sebelumnya sudah tersimpan."
		}
		if ctx.Err() != nil {
			category = "canceled"
			message = "Persiapan terhenti. Materi tersimpan dapat dilanjutkan."
		}
		fail(message, category)
		return
	}
	if err = lesson.Validate(); err != nil {
		fail("Materi belum sesuai pilihan. Coba lanjutkan kembali.", "validation")
		return
	}
	if checkpoints["final-lesson"] == nil {
		raw, e := json.Marshal(lesson)
		if e == nil {
			e = s.jobs.SaveCheckpoint(ctx, uid, id, "final-lesson", raw, 95)
		}
		if e != nil {
			fail("Materi belum berhasil disimpan. Coba lanjutkan kembali.", "storage")
			return
		}
	}
	saved, err := s.lessons.Create(ctx, uid, job.RequestID, lesson)
	if err != nil {
		fail("Materi belum berhasil disimpan. Coba lanjutkan kembali.", "storage")
		return
	}
	if err = s.jobs.Complete(ctx, uid, id, saved.ID); err != nil {
		fail("Sesi belum berhasil dituntaskan. Coba lanjutkan kembali.", "storage")
		return
	}
	slog.Info("preparation complete", "job", id, "lesson", saved.ID)
}
