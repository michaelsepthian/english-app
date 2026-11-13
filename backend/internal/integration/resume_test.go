package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"english-app/internal/handler"
	"english-app/internal/model"
	"english-app/internal/platform/identity"
	"english-app/internal/repository"
	"english-app/internal/repository/postgres"
	"english-app/internal/service"
)

func resumeStore(t *testing.T) *testStore {
	t.Helper()
	if os.Getenv("RUN_DB_TESTS") != "1" {
		t.Skip("opt-in PostgreSQL tests")
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")), Host: os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"), Path: os.Getenv("DB_NAME"), RawQuery: "sslmode=disable"}
	store, err := NewStore(t.Context(), u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Pool.Close)
	return store
}
func resumeUser(t *testing.T, store *testStore) model.User {
	t.Helper()
	uid := identity.NewID()
	u := model.User{ID: uid, Email: uid + "@example.test", Settings: model.DefaultSettings("Resume Tester")}
	if err := postgres.New(store.Pool).Users.Create(t.Context(), u, "unused-password"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = store.Pool.Exec(context.Background(), "DELETE FROM english_app.users WHERE id=$1", uid) })
	return u
}
func waitStudyJob(t *testing.T, jobs repository.Jobs, uid, id, status string) *model.StudyJob {
	t.Helper()
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
		j, err := jobs.Find(t.Context(), uid, id)
		if err != nil {
			t.Fatal(err)
		}
		if j != nil && j.Status == status {
			return j
		}
	}
	t.Fatalf("job never reached %s", status)
	return nil
}

type resumeAI struct {
	fakeAI
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
	want    model.Settings
	t       *testing.T
}

func (f *resumeAI) GeneratePlanCheckpointed(ctx context.Context, s model.Settings, p model.StudyPlan, h model.PreparationHooks) (model.Lesson, error) {
	f.calls.Add(1)
	if s.Goal != f.want.Goal || s.Level != f.want.Level {
		f.t.Error("snapshot changed")
	}
	if _, ok := ctx.Deadline(); ok {
		f.t.Error("global worker deadline remains")
	}
	if f.entered != nil {
		close(f.entered)
		select {
		case <-f.release:
		case <-ctx.Done():
			return model.Lesson{}, ctx.Err()
		}
	}
	if h.BeforeAttempt != nil {
		if err := h.BeforeAttempt("fixture", "Menyimpan materi", 90, 1); err != nil {
			return model.Lesson{}, err
		}
	}
	return plannedFixture(p), nil
}

type lessonSaveFailure struct {
	repository.Lessons
	fail atomic.Bool
}

func (r *lessonSaveFailure) Create(ctx context.Context, u, id string, l model.Lesson) (*model.LessonRecord, error) {
	if r.fail.Swap(false) {
		return nil, errors.New("injected final storage failure")
	}
	return r.Lessons.Create(ctx, u, id, l)
}

func TestResumeFinalSaveSnapshotAndHTTPIsolation(t *testing.T) {
	store := resumeStore(t)
	user := resumeUser(t, store)
	other := resumeUser(t, store)
	repos := postgres.New(store.Pool)
	failures := &lessonSaveFailure{Lessons: repos.Lessons}
	failures.fail.Store(true)
	repos.Lessons = failures
	ai := &resumeAI{want: user.Settings, t: t}
	services := service.New(repos, ai, nil, service.Config{AIConfigured: true, DailyLimit: 120})
	t.Cleanup(func() { _ = services.Studies.Shutdown(context.Background()) })
	plan := model.StudyPlan{Reading: 1, Listening: 1, Questions: 2}
	job, _, err := services.Studies.Prepare(t.Context(), user.ID, model.PrepareRequest{RequestID: identity.NewID(), Plan: plan})
	if err != nil {
		t.Fatal(err)
	}
	failed := waitStudyJob(t, repos.Jobs, user.ID, job.ID, "failed")
	if !failed.CanResume || failed.Progress != 95 {
		t.Fatalf("failed job lost progress %+v", failed)
	}
	parts, err := repos.Jobs.Checkpoints(t.Context(), user.ID, job.ID)
	if err != nil || parts["final-lesson"] == nil {
		t.Fatal("final checkpoint missing", err)
	}
	foreign, err := repos.Jobs.Checkpoints(t.Context(), other.ID, job.ID)
	if err != nil || len(foreign) != 0 {
		t.Fatal("cross-user checkpoint leak")
	}
	// Settings changed after failure must never affect a resumed job.
	changed := user.Settings
	changed.Goal = "New goal after failure"
	changed.Level = "B2"
	if err = repos.Users.UpdateSettings(t.Context(), user.ID, changed); err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer((&handler.Server{Services: services, Config: handler.Config{Origin: "http://localhost:5180"}}).Handler())
	defer host.Close()
	token := identity.HashToken(identity.NewID())
	if err = repos.Sessions.Create(t.Context(), user.ID, identity.HashToken(token), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	otherToken := identity.HashToken(identity.NewID())
	if err = repos.Sessions.Create(t.Context(), other.ID, identity.HashToken(otherToken), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	post := func(token string) (int, string) {
		raw, _ := json.Marshal(map[string]string{"id": job.ID})
		req, _ := http.NewRequest("POST", host.URL+"/api/lesson/job/resume", bytes.NewReader(raw))
		req.Header.Set("Origin", "http://localhost:5180")
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "english_session", Value: token})
		}
		resp, e := host.Client().Do(req)
		if e != nil {
			t.Error(e)
			return 0, ""
		}
		defer resp.Body.Close()
		var out bytes.Buffer
		_, _ = out.ReadFrom(resp.Body)
		return resp.StatusCode, out.String()
	}
	if code, _ := post(""); code != 401 {
		t.Fatalf("unauthenticated resume %d", code)
	}
	if code, _ := post(otherToken); code != 404 {
		t.Fatalf("foreign resume %d", code)
	}
	code, body := post(token)
	if code != 202 {
		t.Fatalf("resume %d %s", code, body)
	}
	if strings.Contains(body, "\"answer\"") || strings.Contains(body, "\"transcript\"") || strings.Contains(body, "final-lesson") {
		t.Fatal("private checkpoint in response")
	}
	done := waitStudyJob(t, repos.Jobs, user.ID, job.ID, "done")
	if ai.calls.Load() != 1 || done.Goal != user.Goal || done.Level != user.Level {
		t.Fatal("resume regenerated AI or changed snapshot")
	}
	var count int
	if err = store.Pool.QueryRow(t.Context(), "SELECT count(*) FROM english_app.lessons WHERE user_id=$1 AND request_id=$2", user.ID, job.RequestID).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate final lesson", err, count)
	}
}
func TestResumeConcurrentAndStartupRecovery(t *testing.T) {
	store := resumeStore(t)
	u := resumeUser(t, store)
	repos := postgres.New(store.Pool)
	p := model.StudyPlan{Reading: 1, Questions: 1}
	if err := repos.Jobs.Create(t.Context(), u.ID, identity.NewID(), p, u.Settings); err != nil {
		t.Fatal(err)
	}
	j, err := repos.Jobs.Find(t.Context(), u.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if ok, e := repos.Jobs.Claim(t.Context(), u.ID, j.ID); e != nil || !ok {
		t.Fatal("claim", e)
	}
	if err = repos.Jobs.SaveCheckpoint(t.Context(), u.ID, j.ID, "saved-part", json.RawMessage(`{"private":"checkpoint"}`), 47); err != nil {
		t.Fatal(err)
	}
	if err = repos.Jobs.RecoverInterrupted(t.Context()); err != nil {
		t.Fatal(err)
	}
	failed, _ := repos.Jobs.Find(t.Context(), u.ID, j.ID)
	if !failed.CanResume || failed.Progress != 47 {
		t.Fatal("startup lost resume progress")
	}
	ai := &resumeAI{want: u.Settings, t: t, entered: make(chan struct{}), release: make(chan struct{})}
	svc := service.New(repos, ai, nil, service.Config{AIConfigured: true, DailyLimit: 120})
	t.Cleanup(func() { _ = svc.Studies.Shutdown(context.Background()) })
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			got, e := svc.Studies.Resume(t.Context(), u.ID, j.ID)
			if e != nil || got.ID != j.ID {
				t.Errorf("concurrent resume %v %+v", e, got)
			}
		})
	}
	wg.Wait()
	<-ai.entered
	active, _ := svc.Studies.Resume(t.Context(), u.ID, j.ID)
	if active.ID != j.ID {
		t.Fatal("active resume returned different job")
	}
	parts, _ := repos.Jobs.Checkpoints(t.Context(), u.ID, j.ID)
	if len(parts) != 1 {
		t.Fatal("checkpoint lost on resume")
	}
	close(ai.release)
	waitStudyJob(t, repos.Jobs, u.ID, j.ID, "done")
	if ai.calls.Load() != 1 {
		t.Fatalf("concurrent workers %d", ai.calls.Load())
	}
}
func TestCheckpointWriteRollsBackProgress(t *testing.T) {
	store := resumeStore(t)
	u := resumeUser(t, store)
	repos := postgres.New(store.Pool)
	p := model.StudyPlan{Reading: 1, Questions: 1}
	if err := repos.Jobs.Create(t.Context(), u.ID, identity.NewID(), p, u.Settings); err != nil {
		t.Fatal(err)
	}
	j, _ := repos.Jobs.Find(t.Context(), u.ID, "")
	_, _ = repos.Jobs.Claim(t.Context(), u.ID, j.ID)
	if err := repos.Jobs.SaveCheckpoint(t.Context(), u.ID, j.ID, "bad", json.RawMessage(`not-json`), 95); err == nil {
		t.Fatal("bad checkpoint saved")
	}
	after, _ := repos.Jobs.Find(t.Context(), u.ID, j.ID)
	parts, _ := repos.Jobs.Checkpoints(t.Context(), u.ID, j.ID)
	if after.Progress != 0 || len(parts) != 0 {
		t.Fatal("partial checkpoint transaction committed")
	}
}
