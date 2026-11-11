// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"net/http"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
)

func (s *Server) wordAPI(w http.ResponseWriter, r *http.Request, u model.User, route string) error {
	switch r.Method + " " + route {
	case "GET words":
		words, err := s.Services.Words.List(r.Context(), u.ID)
		if err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"words": words})
	case "POST words/lookup":
		var in model.LookupRequest
		if err := decode(w, r, &in); err != nil {
			return err
		}
		word, words, err := s.Services.Words.Lookup(r.Context(), u.ID, in.Query)
		if err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"words": words, "word": word})
	default:
		return apperror.NewError(404, "Halaman API tidak ditemukan.")
	}
	return nil
}
