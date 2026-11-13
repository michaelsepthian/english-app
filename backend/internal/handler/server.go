// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"english-app/internal/platform/apperror"
)

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(self)")
		if r.URL.Path == "/healthz" {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			if s.Services.Health.Ping(ctx) != nil {
				http.Error(w, "database unavailable", 503)
				return
			}
			writeJSON(w, 200, map[string]any{"status": "ok"})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			ctx, cancel := context.WithTimeout(r.Context(), 100*time.Second)
			defer cancel()
			r = r.WithContext(ctx)
			if err := s.api(w, r); err != nil {
				s.fail(w, err)
			}
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "method not allowed", 405)
			return
		}
		path := filepath.Join(s.Config.StaticDir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			http.ServeFile(w, r, path)
			return
		}
		if strings.Contains(filepath.Base(r.URL.Path), ".") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(s.Config.StaticDir, "index.html"))
	})
}

func (s *Server) api(w http.ResponseWriter, r *http.Request) error {
	route := strings.TrimPrefix(r.URL.Path, "/api/")
	method := r.Method
	ctx := r.Context()
	if method != "GET" && method != "HEAD" {
		origin := r.Header.Get("Origin")
		expected := s.Config.Origin
		if expected == "" {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			expected = scheme + "://" + r.Host
		}
		if origin != expected || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			return apperror.NewError(403, "Permintaan lintas situs ditolak.")
		}
	}
	token := ""
	if cookie, err := r.Cookie("english_session"); err == nil {
		token = cookie.Value
	}
	// Public account routes run after origin checks, before requiring a session.
	switch method + " " + route {
	case "POST auth/register", "POST auth/login", "POST auth/logout", "GET auth/me":
		return s.authAPI(w, r, token, route)
	}

	u, err := s.Services.Auth.Session(ctx, token)
	if err != nil {
		return err
	}
	uid := u.ID
	if route == "lesson/prepare" || route == "lesson/job" || route == "lesson/job/resume" || route == "lesson/review" {
		return s.studyAPI(w, r, uid, route)
	}
	if strings.HasPrefix(route, "reading/") {
		return s.readingAPI(w, r, uid, route)
	}
	// Authenticated routing: each feature owns its request handling.
	switch method + " " + route {
	case "POST ai/test":
		return s.providerAPI(w, r, u, route)
	case "GET dashboard", "PATCH settings", "GET calendar":
		return s.dashboardAPI(w, r, u, route)
	case "GET words", "POST words/lookup":
		return s.wordAPI(w, r, u, route)
	case "GET lesson/history", "GET lesson", "POST lesson/generate", "PATCH lesson/draft", "POST lesson/submit", "POST lesson/heartbeat":
		return s.lessonAPI(w, r, u, route)
	case "GET chat", "POST chat":
		return s.chatAPI(w, r, u, route)
	case "POST audio":
		return s.audioAPI(w, r, u, route)
	default:
		return apperror.NewError(404, "Halaman API tidak ditemukan.")
	}
}
