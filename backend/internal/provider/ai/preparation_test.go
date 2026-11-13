package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func completionBody(text string) io.ReadCloser {
	raw, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": text}, "finish_reason": "stop"}}})
	return io.NopCloser(strings.NewReader(string(raw)))
}
func preparationParts() []string {
	passage := strings.Repeat("We practice English together at work every morning. ", 5)
	qs := []model.Question{}
	for i, id := range []string{"reading-1", "reading-2"} {
		qs = append(qs, model.Question{ID: []string{"q1", "q2"}[i], Skill: "reading", SourceID: id, Prompt: "When do we practice?", Options: []string{"Morning", "Evening", "Night", "Afternoon"}, Answer: 0, Explanation: "Bacaan menyebutkan bahwa latihan dilakukan pada pagi hari bersama rekan kerja."})
	}
	raw, _ := json.Marshal(map[string]any{"questions": qs})
	return []string{`{"title":"First","passage":"` + passage + `"}`, `{"title":"Second","passage":"` + passage + `"}`, string(raw)}
}
func TestPreparationRetryAndResume(t *testing.T) {
	for _, failures := range []int{1, 2, 3} {
		t.Run(string(rune('0'+failures)), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx := t.Context()
				parts := preparationParts()
				calls := []string{}
				attempts := []int{}
				saved := map[string]json.RawMessage{}
				stageCalls := map[string]int{}
				firstInput := ""
				a := NewAI("test", "model")
				a.BaseURL = "https://test.invalid"
				a.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
					raw, _ := io.ReadAll(r.Body)
					input := string(raw)
					calls = append(calls, input)
					index := len(saved)
					stageCalls[[]string{"first", "second", "questions"}[index]]++
					if index == 1 && stageCalls["second"] <= failures {
						if firstInput == "" {
							firstInput = input
						} else if firstInput != input {
							t.Error("retry changed input")
						}
						return &http.Response{StatusCode: 504, Body: io.NopCloser(strings.NewReader("timeout")), Header: http.Header{}}, nil
					}
					return &http.Response{StatusCode: 200, Body: completionBody(parts[index]), Header: http.Header{}}, nil
				})
				hooks := model.PreparationHooks{Checkpoints: saved, BeforeAttempt: func(key, stage string, p, n int) error { attempts = append(attempts, n); return nil }, Save: func(key string, raw json.RawMessage, p int) error {
					saved[key] = append(json.RawMessage{}, raw...)
					return nil
				}}
				p := model.StudyPlan{Reading: 2, Questions: 2}
				_, err := a.GeneratePlanCheckpointed(ctx, model.DefaultSettings("Test"), p, hooks)
				if failures < 3 {
					if err != nil || len(saved) != 3 || len(calls) != 3+failures {
						t.Fatalf("retry failed: %v %d %d", err, len(saved), len(calls))
					}
				} else {
					if !apperror.IsTimeout(err) || len(saved) != 1 || stageCalls["second"] != 3 {
						t.Fatalf("bad exhaustion %v %+v", err, stageCalls)
					}
					// A resumed run starts a new attempt budget and skips the saved first reading.
					stageCalls["second"] = failures
					_, err = a.GeneratePlanCheckpointed(ctx, model.DefaultSettings("Test"), p, hooks)
					if err != nil || len(saved) != 3 || stageCalls["first"] != 1 || attempts[len(attempts)-2] != 1 {
						t.Fatalf("resume failed %v %+v", err, stageCalls)
					}
				}
			})
		})
	}
}
func TestPreparationDoesNotRetryOtherFailures(t *testing.T) {
	for _, status := range []int{400, 401, 402, 403, 429, 500, 502, 408} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				a := NewAI("test", "model")
				a.BaseURL = "https://test.invalid"
				calls := 0
				a.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("error")), Header: http.Header{}}, nil
				})
				_, err := a.GeneratePlanCheckpointed(t.Context(), model.DefaultSettings("Test"), model.StudyPlan{Reading: 1, Questions: 1}, model.PreparationHooks{})
				expected := 1
				if status == 408 {
					expected = 3
				}
				if err == nil || calls != expected {
					t.Fatalf("status %d calls=%d error=%v", status, calls, err)
				}
			})
		})
	}
	for _, response := range []string{"invalid JSON", `{"title":"Invalid","passage":"short"}`} {
		t.Run(response, func(t *testing.T) {
			a := NewAI("test", "model")
			a.BaseURL = "https://test.invalid"
			calls := 0
			saves := 0
			a.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Body: completionBody(response), Header: http.Header{}}, nil
			})
			_, err := a.GeneratePlanCheckpointed(t.Context(), model.DefaultSettings("Test"), model.StudyPlan{Reading: 1, Questions: 1}, model.PreparationHooks{Save: func(string, json.RawMessage, int) error { saves++; return nil }})
			if err == nil || calls != 1 || saves != 0 {
				t.Fatalf("invalid material persisted or retried %v %d %d", err, calls, saves)
			}
		})
	}
}
func TestPreparationCheckpointFailureAndQuota(t *testing.T) {
	for _, quotaFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "storage", true: "quota"}[quotaFailure], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				a := NewAI("test", "model")
				a.BaseURL = "https://test.invalid"
				calls := 0
				checked := 0
				sentinel := errors.New("stop")
				a.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls++
					if quotaFailure {
						return &http.Response{StatusCode: 408, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
					}
					return &http.Response{StatusCode: 200, Body: completionBody(preparationParts()[0]), Header: http.Header{}}, nil
				})
				_, err := a.GeneratePlanCheckpointed(t.Context(), model.DefaultSettings("Test"), model.StudyPlan{Reading: 2, Questions: 2}, model.PreparationHooks{
					BeforeAttempt: func(string, string, int, int) error {
						checked++
						if quotaFailure && checked == 2 {
							return sentinel
						}
						return nil
					},
					Save: func(string, json.RawMessage, int) error { return sentinel },
				})
				if !errors.Is(err, sentinel) || calls != 1 {
					t.Fatalf("continued after failure %v calls=%d", err, calls)
				}
			})
		})
	}
}
func TestPreparationCancelDuringBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		a := NewAI("test", "model")
		a.BaseURL = "https://test.invalid"
		calls := 0
		a.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 408, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
		})
		go func() { time.Sleep(time.Second); cancel() }()
		_, err := a.GeneratePlanCheckpointed(ctx, model.DefaultSettings("Test"), model.StudyPlan{Reading: 1, Questions: 1}, model.PreparationHooks{})
		if !errors.Is(err, context.Canceled) || calls != 1 {
			t.Fatalf("cancel ignored %v calls=%d", err, calls)
		}
	})
}

type timedOut struct{}

func (timedOut) Error() string { return "private transport details" }
func (timedOut) Timeout() bool { return true }

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, timedOut{} }
func (failingBody) Close() error             { return nil }
func TestRequestRetainsTimeoutCause(t *testing.T) {
	for _, reading := range []bool{false, true} {
		a := NewAI("test", "model")
		a.BaseURL = "https://test.invalid"
		a.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			if reading {
				return &http.Response{StatusCode: 200, Body: failingBody{}, Header: http.Header{}}, nil
			}
			return nil, timedOut{}
		})
		_, err := a.request(t.Context(), "/chat/completions", "application/json", strings.NewReader("{}"))
		if !apperror.IsTimeout(err) || strings.Contains(err.Error(), "private") {
			t.Fatalf("cause missing or leaked: %v", err)
		}
		var cause timedOut
		if !errors.As(err, &cause) {
			t.Fatal("transport cause lost")
		}
	}
}
