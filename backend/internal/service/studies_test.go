package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"english-app/internal/model"
	"english-app/internal/repository"
)

type workerJobs struct {
	repository.Jobs
	parts                map[string]json.RawMessage
	done, failed         bool
	saveErr, completeErr bool
}

func (r *workerJobs) Claim(context.Context, string, string) (bool, error) { return true, nil }
func (r *workerJobs) Checkpoints(context.Context, string, string) (map[string]json.RawMessage, error) {
	return r.parts, nil
}
func (r *workerJobs) SaveCheckpoint(_ context.Context, _, _, key string, raw json.RawMessage, _ int) error {
	if r.saveErr {
		return errors.New("checkpoint storage")
	}
	r.parts[key] = raw
	return nil
}
func (r *workerJobs) Fail(context.Context, string, string, string) error { r.failed = true; return nil }
func (r *workerJobs) Complete(context.Context, string, string, string) error {
	if r.completeErr {
		return errors.New("complete storage")
	}
	r.done = true
	return nil
}

type workerLessons struct {
	repository.Lessons
	calls int
	err   bool
}

func (r *workerLessons) Create(_ context.Context, _, _ string, l model.Lesson) (*model.LessonRecord, error) {
	r.calls++
	if r.err {
		return nil, errors.New("lesson storage")
	}
	return &model.LessonRecord{ID: "saved", Content: l}, nil
}

type workerAI struct {
	AI
	generate func(context.Context) (model.Lesson, error)
	calls    int
}

func (a *workerAI) GeneratePlanCheckpointed(ctx context.Context, _ model.Settings, _ model.StudyPlan, _ model.PreparationHooks) (model.Lesson, error) {
	a.calls++
	return a.generate(ctx)
}
func workerFixture() model.Lesson {
	p := model.StudyPlan{Reading: 1, Questions: 1}
	return model.Lesson{Title: "Workplace English", Plan: &p, Readings: []model.ReadingItem{{ID: "reading-1", Reading: model.Reading{Title: "Practice", Passage: strings.Repeat("We practice English at work every morning. ", 5)}}}, Questions: []model.Question{{ID: "q1", SourceID: "reading-1", Skill: "reading", Prompt: "When do we practice?", Options: []string{"Morning", "Evening", "Night", "Afternoon"}, Answer: 0, Explanation: "Bacaan menyebutkan waktu latihan pada pagi hari bersama rekan kerja."}}}
}
func workerService(a *workerAI, j *workerJobs, l *workerLessons) *StudyService {
	ctx, cancel := context.WithCancel(context.Background())
	return &StudyService{jobs: j, lessons: l, tutor: &TutorService{ai: a, configured: true}, workers: make(chan struct{}, 2), workerContext: ctx, cancelWorkers: cancel}
}
func workerJob() *model.StudyJob {
	s := model.DefaultSettings("Test")
	return &model.StudyJob{ID: "job", RequestID: "request", Settings: &s, Plan: *workerFixture().Plan}
}
func TestWorkerHasNoTwelveMinuteDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		a := &workerAI{generate: func(ctx context.Context) (model.Lesson, error) {
			if _, deadline := ctx.Deadline(); deadline {
				t.Error("global deadline present")
			}
			// Simulate ten successful 89-second AI requests without wall-clock delay.
			for range 10 {
				time.Sleep(89 * time.Second)
				if ctx.Err() != nil {
					return model.Lesson{}, ctx.Err()
				}
			}
			return workerFixture(), nil
		}}
		j := &workerJobs{parts: map[string]json.RawMessage{}}
		l := &workerLessons{}
		s := workerService(a, j, l)
		defer s.cancelWorkers()
		started := time.Now()
		s.run("owner", workerJob())
		if !j.done || j.failed || time.Since(started) <= 12*time.Minute {
			t.Fatalf("long preparation stopped done=%v failed=%v", j.done, j.failed)
		}
	})
}
func TestWorkerStopsOnCheckpointAndFinalStorageFailures(t *testing.T) {
	for _, failure := range []string{"checkpoint", "lesson", "completion"} {
		t.Run(failure, func(t *testing.T) {
			a := &workerAI{generate: func(context.Context) (model.Lesson, error) { return workerFixture(), nil }}
			j := &workerJobs{parts: map[string]json.RawMessage{}, saveErr: failure == "checkpoint", completeErr: failure == "completion"}
			l := &workerLessons{err: failure == "lesson"}
			s := workerService(a, j, l)
			defer s.cancelWorkers()
			s.run("owner", workerJob())
			if !j.failed || j.done {
				t.Fatal("storage failure not marked failed")
			}
			if failure == "checkpoint" {
				if l.calls != 0 {
					t.Fatal("continued after failed checkpoint")
				}
				return
			}
			j.failed = false
			j.completeErr = false
			l.err = false
			s.run("owner", workerJob())
			if !j.done || a.calls != 1 {
				t.Fatal("final save resume called AI again")
			}
		})
	}
}
func TestShutdownCancelsWorker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entered := make(chan struct{})
		a := &workerAI{generate: func(ctx context.Context) (model.Lesson, error) {
			close(entered)
			<-ctx.Done()
			return model.Lesson{}, ctx.Err()
		}}
		j := &workerJobs{parts: map[string]json.RawMessage{}}
		s := workerService(a, j, &workerLessons{})
		s.launch("owner", workerJob())
		<-entered
		if err := s.Shutdown(t.Context()); err != nil {
			t.Fatal(err)
		}
		if !j.failed || j.done {
			t.Fatal("shutdown did not preserve resumable failure")
		}
	})
}
