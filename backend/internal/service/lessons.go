// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"context"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/repository"
)

type LessonService struct {
	repo   repository.Lessons
	tutor  *TutorService
	limits *Limiter
}

func (s *LessonService) Get(ctx context.Context, uid, id string) (*model.LessonRecord, error) {
	return s.repo.Find(ctx, uid, id)
}
func (s *LessonService) History(ctx context.Context, uid string) ([]model.LessonSummary, error) {
	rows, err := s.repo.History(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := []model.LessonSummary{}
	for _, r := range rows {
		v := model.LessonSummary{ID: r.ID, Day: r.Day, Title: r.Title, Total: r.Total, Completed: r.Attempt != nil}
		if r.Attempt != nil {
			v.Percent = model.Percent(*r.Attempt)
		}
		out = append(out, v)
	}
	return out, nil
}
func (s *LessonService) Generate(ctx context.Context, u model.User) (*model.LessonRecord, error) {
	lesson, err := s.repo.Find(ctx, u.ID, "")
	if err != nil || lesson != nil {
		return lesson, err
	}
	ai, err := s.tutor.Client()
	if err != nil {
		return nil, err
	}
	release, err := s.limits.Lock(ctx, "lesson:"+u.ID)
	if err != nil {
		return nil, err
	}
	defer release()
	if err = s.tutor.quota(ctx, u.ID); err != nil {
		return nil, err
	}
	recent, err := s.repo.RecentThemes(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	generated, err := ai.Generate(ctx, u.Settings, recent)
	if err != nil {
		return nil, err
	}
	if err = generated.Validate(); err != nil {
		return nil, apperror.NewError(502, err.Error())
	}
	return s.repo.Create(ctx, u.ID, "legacy-"+model.DayKey(time.Now()), generated)
}
func (s *LessonService) Draft(ctx context.Context, uid, id string, answers map[string]int) error {
	if err := validateAnswers(id, answers); err != nil {
		return err
	}
	l, err := s.repo.Find(ctx, uid, id)
	if err != nil {
		return err
	}
	if l == nil {
		return apperror.NewError(404, "Pelajaran tidak ditemukan.")
	}
	if l.Attempt != nil {
		return apperror.NewError(409, "Pelajaran sudah selesai.")
	}
	for key, v := range answers {
		found := false
		for _, q := range l.Content.Questions {
			if q.ID == key {
				found = true
			}
		}
		if !found || v < 0 || v > 3 {
			return apperror.NewError(400, "Jawaban tidak valid.")
		}
	}
	return s.repo.SaveDraft(ctx, uid, id, answers)
}

func (s *LessonService) Heartbeat(ctx context.Context, uid, id string, active bool) error {
	if !idPattern.MatchString(id) {
		return apperror.NewError(400, "Pelajaran belum valid.")
	}
	changed, err := s.repo.Heartbeat(ctx, uid, id, active)
	if err != nil {
		return err
	}

	if !changed {
		l, err := s.repo.Find(ctx, uid, id)
		if err != nil {
			return err
		}
		if l == nil {
			return apperror.NewError(404, "Pelajaran tidak ditemukan.")
		}
	}
	return nil
}

func (s *LessonService) Submit(ctx context.Context, uid, id string, answers map[string]int) (*model.LessonRecord, error) {
	if err := validateAnswers(id, answers); err != nil {
		return nil, err
	}
	l, err := s.repo.Find(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, apperror.NewError(404, "Pelajaran tidak ditemukan.")
	}
	if l.Attempt != nil {
		return l, nil
	}
	if _, err = model.Grade(l.Content, answers, 0); err != nil {
		return nil, apperror.NewError(400, err.Error())
	}
	if err = s.Heartbeat(ctx, uid, id, false); err != nil {
		return nil, err
	}
	l, err = s.repo.Find(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	attempt, err := model.Grade(l.Content, answers, l.Seconds)
	if err != nil {
		return nil, apperror.NewError(400, err.Error())
	}
	err = s.repo.CreateAttempt(ctx, uid, id, attempt)
	if err != nil {
		return nil, err
	}
	return s.repo.Find(ctx, uid, id)
}
