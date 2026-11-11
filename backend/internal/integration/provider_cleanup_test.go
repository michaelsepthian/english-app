package integration

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"english-app/internal/platform/apperror"
	"english-app/internal/provider/ai"
	"english-app/internal/repository"
	"english-app/internal/service"
)

func TestAPIKeyResponsesCannotForwardCredentialsThroughRedirect(t *testing.T) {
	var reached atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Store(true) }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" || r.Header.Get("Authorization") != "Bearer selected-api-key" {
			t.Error("wrong API endpoint or credential")
		}
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	client := ai.NewAI("selected-api-key", "selected-model")
	if err := client.ConfigureCompatible(source.URL, "responses", "prompt"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Probe(t.Context()); err == nil {
		t.Fatal("redirect accepted as a model response")
	}
	if reached.Load() {
		t.Fatal("API key was forwarded to a redirect target")
	}
}

func TestUnconfiguredAPIClientFailsWithoutSubscriptionFallback(t *testing.T) {
	server := service.New(repository.Repositories{}, nil, nil, service.Config{AIConfigured: true})
	_, err := server.Tutor.Client()
	var apiErr *apperror.AppError
	if !errors.As(err, &apiErr) || apiErr.Status != 503 {
		t.Fatalf("missing client accepted: %v", err)
	}
}
