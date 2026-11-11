// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"io"
	"net/http"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
)

func (s *Server) dashboardAPI(w http.ResponseWriter, r *http.Request, u model.User, route string) error {
	ctx := r.Context()
	uid := u.ID
	var err error
	switch r.Method + " " + route {
	case "GET dashboard":
		stats, err := s.Services.Progress.Weekly(ctx, uid)
		if err != nil {
			return err
		}
		lesson, err := s.Services.Lessons.Get(ctx, uid, r.URL.Query().Get("id"))
		if err != nil {
			return err
		}
		var public any
		if lesson != nil {
			public = lesson.Public()
		}
		writeJSON(w, 200, map[string]any{
			"user": u, "stats": stats, "lesson": public, "aiConfigured": s.Services.Tutor.Configured(),
			"pronunciationAvailable": s.Services.Reading.Available(),
		})
	case "PATCH settings":
		var in model.Settings
		if err = decode(w, r, &in); err != nil {
			return err
		}
		u, err = s.Services.Profile.Update(ctx, u, in)
		if err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"user": u})
	case "GET calendar":
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="english-garden.ics"`)
		_, _ = io.WriteString(w, model.Calendar(u.Settings, uid, s.Config.Origin, time.Now()))
	default:
		return apperror.NewError(404, "Halaman API tidak ditemukan.")
	}
	return nil
}
