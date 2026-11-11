package integration

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/platform/jsonutil"
	"english-app/internal/provider/ai"
	"english-app/internal/repository"
	"english-app/internal/service"
)

func completion(w http.ResponseWriter, text, finish string) {
	writeJSON(
		w,
		200,
		map[string]any{"choices": []any{map[string]any{"finish_reason": finish, "message": map[string]any{"role": "assistant", "content": text}}}},
	)
}
func TestCompatibleLessonDictionaryAndChat(t *testing.T) {
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer provider-key" {
			t.Error("wrong endpoint or credential")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Fatal("invalid request")
		}
		if body["model"] != "sonnet-5" || body["stream"] != false {
			t.Error("wrong model or stream mode")
		}
		if _, ok := body["input"]; ok {
			t.Error("Responses input sent to Chat Completions")
		}
		if _, ok := body["max_output_tokens"]; ok {
			t.Error("wrong output limit field")
		}
		if _, ok := body["response_format"]; ok {
			t.Error("prompt mode must not require vendor-specific JSON support")
		}
		messages := body["messages"].([]any)
		if messages[0].(map[string]any)["role"] != "system" {
			t.Error("system prompt missing")
		}
		switch requests {
		case 1:
			if body["max_tokens"] != float64(6500) {
				t.Error("lesson limit missing")
			}
			completion(w, "```json\n"+string(jsonutil.Marshal(fixtureLesson()))+"\n```", "stop")
		case 2:
			completion(w, string(jsonutil.Marshal(model.StarterWords[0])), "stop")
		case 3:
			if len(messages) != 12 || body["max_tokens"] != float64(450) {
				t.Error("chat history/output not bounded")
			}
			completion(w, "Hello! How was your day?", "stop")
		case 4:
			if body["max_tokens"] != float64(16) {
				t.Error("connection test must be short")
			}
			completion(w, "OK", "stop")
		}
	}))
	defer upstream.Close()
	a := ai.NewAI("provider-key", "sonnet-5")
	if err := a.ConfigureCompatible(upstream.URL+"/v1/", "chat_completions", "prompt"); err != nil {
		t.Fatal(err)
	}
	l, err := a.Generate(t.Context(), model.DefaultSettings("Test"), nil)
	if err != nil || len(l.Questions) != 8 {
		t.Fatalf("lesson: %v", err)
	}
	word, err := a.Lookup(t.Context(), "go")
	if err != nil || word.V2 != "went" {
		t.Fatalf("dictionary: %v", err)
	}
	history := make([]model.Message, 30)
	for i := range history {
		history[i] = model.Message{Role: "user", Content: "A short message"}
	}
	if reply, err := a.Chat(t.Context(), model.DefaultSettings("Test"), history, "Hello"); err != nil || reply == "" {
		t.Fatalf("chat: %v", err)
	}
	if _, err = a.Probe(t.Context()); err != nil {
		t.Fatal(err)
	}
}
func TestCompatibleErrorAndJSONModes(t *testing.T) {
	for _, mode := range []string{"prompt", "json_object", "json_schema"} {
		t.Run(mode, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if mode != "prompt" {
					format := body["response_format"].(map[string]any)
					if format["type"] != mode {
						t.Error("wrong JSON mode")
					}
				}
				completion(w, string(jsonutil.Marshal(model.StarterWords[0])), "stop")
			}))
			defer upstream.Close()
			a := ai.NewAI("key", "model")
			_ = a.ConfigureCompatible(upstream.URL, "chat_completions", mode)
			if _, err := a.Lookup(t.Context(), "go"); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, test := range []struct {
		name, content, finish string
		status                int
	}{
		{"truncated", `{"word":"go"}`, "length", 200},
		{"invalid JSON", "Here is your answer", "stop", 200},
		{"no choice", "", "", 200},
		{"invalid key", "private-provider-error", "", 401},
		{"empty balance", "private-provider-error", "", 402},
	} {
		t.Run(test.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.status != 200 {
					http.Error(w, test.content, test.status)
					return
				}
				if test.name == "no choice" {
					writeJSON(w, 200, map[string]any{"choices": []any{}})
					return
				}
				completion(w, test.content, test.finish)
			}))
			defer upstream.Close()
			a := ai.NewAI("key", "model")
			_ = a.ConfigureCompatible(upstream.URL, "chat_completions", "prompt")
			_, err := a.Lookup(t.Context(), "go")
			if err == nil {
				t.Fatal("bad response accepted")
			}
			if strings.Contains(err.Error(), "private-provider-error") {
				t.Fatal("upstream diagnostic leaked")
			}
		})
	}
}
func TestCompatibleDoesNotForwardKeyOnRedirect(t *testing.T) {
	var reached atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Store(true) }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer source.Close()
	a := ai.NewAI("secret-key", "model")
	_ = a.ConfigureCompatible(source.URL, "chat_completions", "prompt")
	if _, err := a.Probe(t.Context()); err == nil {
		t.Fatal("redirect treated as completion")
	}
	if reached.Load() {
		t.Fatal("API credential request followed a redirect")
	}
}
func TestCompatibleConfigAndProviderSelection(t *testing.T) {
	for _, base := range []string{
		"http://external.example/v1",
		"https://user:password@example.test/v1",
		"https://example.test/v1?key=secret",
	} {
		a := ai.NewAI("key", "model")
		if a.ConfigureCompatible(base, "chat_completions", "prompt") == nil {
			t.Fatalf("invalid base accepted: %s", base)
		}
	}
	fake := &fakeAI{}
	s := service.New(repository.Repositories{}, fake, nil, service.Config{AIConfigured: true})
	ai, err := s.Tutor.Client()
	if err != nil || ai != fake {
		t.Fatal("configured API-key client was not selected")
	}
	s = service.New(repository.Repositories{}, fake, nil, service.Config{AIConfigured: false})
	_, err = s.Tutor.Client()
	var api *apperror.AppError
	if !errors.As(err, &api) || api.Status != 503 || !strings.Contains(api.Message, "AI_API_KEY") {
		t.Fatal("missing API key not clearly reported")
	}
}
