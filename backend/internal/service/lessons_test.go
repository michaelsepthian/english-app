package service

import (
	"context"
	"errors"
	"testing"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/repository"
)

// Only the persistence operations a lesson workflow should invoke are implemented.
// An unexpected dependency call fails instead of quietly returning a zero value.
type lessonMemory struct {
	repository.Lessons
	lesson                  *model.LessonRecord
	drafts, attempts, ticks int
}

func (r *lessonMemory) Find(_ context.Context, uid, id string) (*model.LessonRecord, error) {
	if uid != "owner" || r.lesson == nil || id != r.lesson.ID {
		return nil, nil
	}
	copy := *r.lesson
	return &copy, nil
}
func (r *lessonMemory) SaveDraft(_ context.Context, uid, id string, a map[string]int) error {
	r.drafts++
	r.lesson.Draft = a
	return nil
}
func (r *lessonMemory) Heartbeat(context.Context, string, string, bool) (bool, error) {
	r.ticks++
	return true, nil
}
func (r *lessonMemory) CreateAttempt(_ context.Context, uid, id string, a model.Attempt) error {
	r.attempts++
	r.lesson.Attempt = &a
	return nil
}
func fixtureRecord() *model.LessonRecord {
	return &model.LessonRecord{ID: "00000000-0000-4000-8000-000000000001", Seconds: 83, Content: model.Lesson{Questions: []model.Question{{ID: "q1", Skill: "vocabulary", Options: []string{"a", "b", "c", "d"}, Answer: 1}, {ID: "q2", Skill: "reading", Options: []string{"a", "b", "c", "d"}, Answer: 2}}}}
}
func assertStatus(t *testing.T, err error, status int) {
	t.Helper()
	var app *apperror.AppError
	if !errors.As(err, &app) || app.Status != status {
		t.Fatalf("want status %d, got %v", status, err)
	}
}
func TestDraftRejectsForeignLessonAndUnknownAnswersBeforeWriting(t *testing.T) {
	r := &lessonMemory{lesson: fixtureRecord()}
	s := &LessonService{repo: r}
	ctx := t.Context()
	id := r.lesson.ID
	assertStatus(t, s.Draft(ctx, "other", id, map[string]int{"q1": 1}), 404)
	assertStatus(t, s.Draft(ctx, "owner", id, map[string]int{"invented": 1}), 400)
	assertStatus(t, s.Draft(ctx, "owner", id, map[string]int{"q1": 4}), 400)
	if r.drafts != 0 {
		t.Fatal("invalid drafts reached persistence")
	}
	if err := s.Draft(ctx, "owner", id, map[string]int{"q1": 1}); err != nil {
		t.Fatal(err)
	}
	if r.drafts != 1 {
		t.Fatal("valid draft was not saved")
	}
	r.lesson.Attempt = &model.Attempt{}
	assertStatus(t, s.Draft(ctx, "owner", id, map[string]int{}), 409)
}
func TestSubmissionKeepsSavedScoreAndTimeOnRetry(t *testing.T) {
	r := &lessonMemory{lesson: fixtureRecord()}
	s := &LessonService{repo: r}
	id := r.lesson.ID
	_, err := s.Submit(t.Context(), "owner", id, map[string]int{"q1": 1})
	assertStatus(t, err, 400)
	if r.ticks != 0 || r.attempts != 0 {
		t.Fatal("incomplete submission changed stored progress")
	}
	l, err := s.Submit(t.Context(), "owner", id, map[string]int{"q1": 1, "q2": 2})
	if err != nil {
		t.Fatal(err)
	}
	if l.Attempt.Score != 2 || l.Attempt.Total != 2 || l.Attempt.Seconds != 83 {
		t.Fatalf("incorrect score/time: %+v", l.Attempt)
	}
	again, err := s.Submit(t.Context(), "owner", id, map[string]int{"q1": 0, "q2": 0})
	if err != nil {
		t.Fatal(err)
	}
	if again.Attempt.Score != 2 || r.attempts != 1 || r.ticks != 1 {
		t.Fatal("retry changed the original result")
	}
}

type attemptMemory struct {
	repository.Lessons
	rows       []model.AttemptRecord
	start, uid string
}

func (r *attemptMemory) Attempts(_ context.Context, uid, start string) ([]model.AttemptRecord, error) {
	r.uid, r.start = uid, start
	return r.rows, nil
}
func TestWeeklyProgressUsesWeightedScoreAndUniqueDays(t *testing.T) {
	r := &attemptMemory{rows: []model.AttemptRecord{{Day: "2026-11-10", Attempt: model.Attempt{Score: 1, Total: 1, Seconds: 65}}, {Day: "2026-11-10", Attempt: model.Attempt{Score: 0, Total: 9, Seconds: 60}}}}
	p, err := (&ProgressService{repo: r}).Weekly(t.Context(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	if p.Accuracy != 10 || p.Seconds != 125 || p.Lessons != 2 || len(p.Days) != 1 || r.uid != "owner" || r.start == "" {
		t.Fatalf("incorrect weekly progress: %+v", p)
	}
}
