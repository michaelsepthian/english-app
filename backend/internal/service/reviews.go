// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"context"
	"sync"

	"english-app/internal/model"
	"english-app/internal/repository"
)

// ReviewService serializes optional feedback within this application instance.
type ReviewService struct {
	lessons repository.Lessons
	users   repository.Users
	tutor   *TutorService
	workers sync.Map
}

func (s *ReviewService) Apply(ctx context.Context, uid string, l *model.LessonRecord) {
	if l.Attempt == nil || l.Attempt.Review != nil {
		return
	}
	key := uid + ":" + l.ID
	if _, busy := s.workers.LoadOrStore(key, true); busy {
		return
	}
	defer s.workers.Delete(key)
	ai, err := s.tutor.Client()
	if err != nil {
		return
	}
	reviewer, ok := ai.(interface {
		Review(context.Context, model.Lesson, model.Attempt, string) (model.LessonReview, error)
	})
	if !ok {
		return
	}
	if s.tutor.quota(ctx, uid) != nil {
		return
	}
	u, err := s.users.FindByID(ctx, uid)
	if err != nil {
		return
	}
	review, err := reviewer.Review(ctx, l.Content, *l.Attempt, u.Goal)
	if err != nil {
		return
	}
	l.Attempt.Review = &review
	_ = s.lessons.SaveReview(ctx, uid, l.ID, *l.Attempt)
}
