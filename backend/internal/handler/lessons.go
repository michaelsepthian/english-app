// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"net/http"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
)

func (s *Server) lessonAPI(w http.ResponseWriter, r *http.Request, u model.User, route string) error {
	ctx, uid := r.Context(), u.ID
	switch r.Method + " " + route {
	case "GET lesson/history":
		history, err := s.Services.Lessons.History(ctx, uid)
		if err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"lessons": history})
	case "GET lesson":
		lesson, err := s.Services.Lessons.Get(ctx, uid, r.URL.Query().Get("id"))
		if err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"lesson": publicLesson(lesson)})
	case "POST lesson/generate":
		var in struct{}
		if err := decode(w, r, &in); err != nil {
			return err
		}
		lesson, err := s.Services.Lessons.Generate(ctx, u)
		if err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"lesson": publicLesson(lesson)})
	case "PATCH lesson/draft", "POST lesson/submit":
		var in model.AnswerRequest
		if err := decode(w, r, &in); err != nil {
			return err
		}
		if route == "lesson/draft" {
			if err := s.Services.Lessons.Draft(ctx, uid, in.ID, in.Answers); err != nil {
				return err
			}
			writeJSON(w, 200, map[string]bool{"ok": true})
			return nil
		}
		lesson, err := s.Services.Lessons.Submit(ctx, uid, in.ID, in.Answers)
		if err != nil {
			return err
		}
		s.Services.Reviews.Apply(ctx, uid, lesson)
		if saved, e := s.Services.Lessons.Get(ctx, uid, in.ID); e == nil && saved != nil {
			lesson = saved
		}
		stats, err := s.Services.Progress.Weekly(ctx, uid)
		if err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"lesson": publicLesson(lesson), "stats": stats})
	case "POST lesson/heartbeat":
		var in model.HeartbeatRequest
		if err := decode(w, r, &in); err != nil {
			return err
		}
		if err := s.Services.Lessons.Heartbeat(ctx, uid, in.ID, in.Active); err != nil {
			return err
		}
		writeJSON(w, 200, map[string]bool{"ok": true})
	default:
		return apperror.NewError(404, "Halaman API tidak ditemukan.")
	}
	return nil
}
func publicLesson(l *model.LessonRecord) any {
	if l == nil {
		return nil
	}
	return l.Public()
}
