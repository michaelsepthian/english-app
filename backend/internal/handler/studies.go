// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"net/http"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
)

func (s *Server) studyAPI(w http.ResponseWriter, r *http.Request, uid, route string) error {
	ctx := r.Context()
	switch r.Method + " " + route {
	case "GET lesson/job":
		job, lesson, err := s.Services.Studies.Job(ctx, uid, r.URL.Query().Get("id"))
		if err != nil {
			return err
		}
		response := map[string]any{"job": job}
		if lesson != nil {
			response["lesson"] = lesson.Public()
		}
		writeJSON(w, 200, response)
	case "POST lesson/prepare":
		var in model.PrepareRequest
		if err := decode(w, r, &in); err != nil {
			return err
		}
		job, accepted, err := s.Services.Studies.Prepare(ctx, uid, in)
		if err != nil {
			return err
		}
		status := 200
		if accepted {
			status = 202
		}
		writeJSON(w, status, map[string]any{"job": job})
	case "POST lesson/job/resume":
		var in model.IDRequest
		if err := decode(w, r, &in); err != nil {
			return err
		}
		job, err := s.Services.Studies.Resume(ctx, uid, in.ID)
		if err != nil {
			return err
		}
		writeJSON(w, 202, map[string]any{"job": job})
	case "POST lesson/review":
		var in model.IDRequest
		if err := decode(w, r, &in); err != nil {
			return err
		}
		lesson, err := s.Services.Lessons.Get(ctx, uid, in.ID)
		if err != nil {
			return err
		}
		if lesson == nil || lesson.Attempt == nil {
			return apperror.NewError(404, "Hasil latihan belum ditemukan.")
		}
		s.Services.Reviews.Apply(ctx, uid, lesson)
		lesson, err = s.Services.Lessons.Get(ctx, uid, in.ID)
		if err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"lesson": publicLesson(lesson)})
	default:
		return apperror.NewError(404, "Halaman belajar tidak ditemukan.")
	}
	return nil
}
