// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"regexp"
	"strings"

	"english-app/internal/platform/apperror"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	var api *apperror.AppError
	if errors.As(err, &api) {
		writeJSON(w, api.Status, map[string]string{"error": api.Message})
		return
	}
	slog.Error("request failed", "type", strings.SplitN(err.Error(), ":", 2)[0])
	writeJSON(w, 500, map[string]string{"error": "Terjadi kesalahan server. Silakan coba lagi."})
}

func decode(w http.ResponseWriter, r *http.Request, v any) error {
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if ct != "application/json" {
		return apperror.NewError(415, "Gunakan format JSON.")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 24*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return apperror.NewError(400, "Isian belum valid. Periksa data yang kamu masukkan.")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return apperror.NewError(400, "JSON belum valid.")
	}
	return nil
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
