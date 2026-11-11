// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"net/http"

	"english-app/internal/model"
)

func (s *Server) providerAPI(w http.ResponseWriter, r *http.Request, u model.User, route string) error {
	var in struct{}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	result, err := s.Services.Tutor.Probe(r.Context(), u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, 200, result)
	return nil
}
