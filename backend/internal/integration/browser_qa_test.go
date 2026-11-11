package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"english-app/internal/model"
	"english-app/internal/platform/identity"
	"english-app/internal/platform/jsonutil"
	"english-app/internal/service"
)

// Opt-in local visual QA with a disposable account and a fake model; never spends API credits.

func TestBrowserQA(t *testing.T) {
	if os.Getenv("RUN_BROWSER_QA") != "1" {
		t.Skip("opt-in browser QA only")
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")),
		Host:   os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"),
		Path:   os.Getenv("DB_NAME"),
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()
	store, err := NewStore(t.Context(), u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Pool.Close()
	id := identity.NewID()
	email := "miva-qa-" + id + "@example.test"
	password, _ := bcrypt.GenerateFromPassword([]byte("Miva-qa-only-2026!"), bcrypt.MinCost)
	settings := model.DefaultSettings("Miva Tester")
	settings.Goal = "Percaya diri mengobrol dalam bahasa Inggris di tempat kerja."
	if _, err = store.Pool.Exec(
		t.Context(),
		"INSERT INTO english_app.users(id,email,password,settings) VALUES($1,$2,$3,$4)",
		id,
		email,
		string(password),
		jsonutil.Marshal(settings),
	); err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = store.Pool.Exec(ctx, "DELETE FROM english_app.users WHERE id=$1", id)
	}()
	server := newTestServer(store, &fakeAI{}, service.Config{AIConfigured: true, Registration: true, DailyLimit: 120})
	server.Config.StaticDir = os.Getenv("STATIC_DIR")
	server.Config.Origin = "http://localhost:5181"
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		t.Fatal(err)
	}
	host := &http.Server{Handler: server.Handler()}
	defer host.Close()
	go host.Serve(listener)
	fmt.Println("QA_EMAIL=" + email)
	deadline := time.Now().Add(15 * time.Minute)
	for time.Now().Before(deadline) {
		if _, err := os.Stat("/src/.qa-stop"); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
}
