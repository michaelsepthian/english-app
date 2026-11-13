// Package repository defines persistence ports; services never depend on pgx or SQL.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"english-app/internal/model"
)

var ErrNotFound = errors.New("record not found")
var ErrConflict = errors.New("record already exists")

type Users interface {
	FindByID(context.Context, string) (model.User, error)
	FindByEmail(context.Context, string) (model.User, string, error)
	Create(context.Context, model.User, string) error
	UpdateSettings(context.Context, string, model.Settings) error
}
type Sessions interface {
	FindUserID(context.Context, string) (string, error)
	Create(context.Context, string, string, time.Time) error
	Delete(context.Context, string) error
}
type Vocabulary interface {
	List(context.Context, string) ([]model.Word, error)
	Upsert(context.Context, string, model.Word) error
}
type Lessons interface {
	Find(context.Context, string, string) (*model.LessonRecord, error)
	History(context.Context, string) ([]model.LessonHistoryRecord, error)
	RecentThemes(context.Context, string) ([]string, error)
	Create(context.Context, string, string, model.Lesson) (*model.LessonRecord, error)
	SaveDraft(context.Context, string, string, map[string]int) error
	Heartbeat(context.Context, string, string, bool) (bool, error)
	CreateAttempt(context.Context, string, string, model.Attempt) error
	Attempts(context.Context, string, string) ([]model.AttemptRecord, error)
	SaveReview(context.Context, string, string, model.Attempt) error
}
type Messages interface {
	List(context.Context, string) ([]model.Message, error)
	CreateExchange(context.Context, string, string, string) error
}
type Jobs interface {
	Find(context.Context, string, string) (*model.StudyJob, error)
	FindByRequest(context.Context, string, string) (*model.StudyJob, error)
	Create(context.Context, string, string, model.StudyPlan, model.Settings) error
	Requeue(context.Context, string, string, model.Settings) (bool, error)
	Checkpoints(context.Context, string, string) (map[string]json.RawMessage, error)
	SaveCheckpoint(context.Context, string, string, string, json.RawMessage, int) error
	Attempt(context.Context, string, string, string, int, int) error
	Claim(context.Context, string, string) (bool, error)
	Progress(context.Context, string, string, string, int) error
	Fail(context.Context, string, string, string) error
	Complete(context.Context, string, string, string) error
	RecoverInterrupted(context.Context) error
}
type Readings interface {
	List(context.Context, string, string) ([]model.ReadingCheck, error)
	Create(context.Context, string, string, model.ReadingCheck) error
}
type Limits interface {
	Increment(context.Context, string, time.Time) (int, error)
	Acquire(context.Context, string, time.Time) (bool, error)
	Release(context.Context, string) error
}
type Health interface{ Ping(context.Context) error }

type Repositories struct {
	Users    Users
	Sessions Sessions
	Words    Vocabulary
	Lessons  Lessons
	Messages Messages
	Jobs     Jobs
	Readings Readings
	Limits   Limits
	Health   Health
}
