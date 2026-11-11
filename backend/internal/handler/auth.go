// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"errors"
	"net"
	"net/http"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
)

func (s *Server) cookie(token string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "english_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.Config.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}
func (s *Server) authAPI(w http.ResponseWriter, r *http.Request, token, route string) error {
	ctx := r.Context()
	switch r.Method + " " + route {
	case "POST auth/register", "POST auth/login":
		var in model.Credentials
		if err := decode(w, r, &in); err != nil {
			return err
		}
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		u, session, err := s.Services.Auth.Authenticate(ctx, in, ip, route == "auth/register")
		if err != nil {
			return err
		}
		http.SetCookie(w, s.cookie(session, 7*86400))
		writeJSON(w, 200, map[string]any{"user": u})
	case "POST auth/logout":
		if err := s.Services.Auth.Logout(ctx, token); err != nil {
			return err
		}
		http.SetCookie(w, s.cookie("", -1))
		writeJSON(w, 200, map[string]bool{"ok": true})
	case "GET auth/me":
		u, err := s.Services.Auth.Session(ctx, token)
		if err != nil {
			var api *apperror.AppError
			if !errors.As(err, &api) {
				return err
			}
			writeJSON(w, 200, map[string]any{"user": nil, "aiConfigured": s.Services.Tutor.Configured()})
			return nil
		}
		writeJSON(w, 200, map[string]any{"user": u, "aiConfigured": s.Services.Tutor.Configured()})
	default:
		return apperror.NewError(404, "Halaman API tidak ditemukan.")
	}
	return nil
}
