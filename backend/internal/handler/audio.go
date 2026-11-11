// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"net/http"

	"english-app/internal/model"
)

func (s *Server) audioAPI(w http.ResponseWriter, r *http.Request, u model.User, route string) error {
	var in model.AudioRequest
	if err := decode(w, r, &in); err != nil {
		return err
	}
	text, err := s.Services.Audio.Text(r.Context(), u.ID, in)
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]string{"text": text, "lang": "en-US", "mode": "device"})
	return nil
}
