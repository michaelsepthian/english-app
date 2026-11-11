// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"net/http"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
)

func (s *Server) chatAPI(w http.ResponseWriter, r *http.Request, u model.User, route string) error {
	var messages []model.Message
	var err error
	switch r.Method + " " + route {
	case "GET chat":
		messages, err = s.Services.Chat.List(r.Context(), u.ID)
	case "POST chat":
		var in model.ChatRequest
		if err = decode(w, r, &in); err != nil {
			return err
		}
		messages, err = s.Services.Chat.Send(r.Context(), u, in.Text)
	default:
		return apperror.NewError(404, "Halaman API tidak ditemukan.")
	}
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"messages": messages})
	return nil
}
