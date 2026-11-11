// MIVA-HTTP: request decoding, session transport and responses.
package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
)

func (s *Server) readingAPI(w http.ResponseWriter, r *http.Request, uid, route string) error {
	ctx := r.Context()
	if r.Method == "GET" && route == "reading/checks" {
		checks, err := s.Services.Reading.List(ctx, uid, r.URL.Query().Get("lessonId"))
		if err != nil {
			return err
		}
		writeJSON(w, 200, map[string]any{"checks": checks})
		return nil
	}

	if r.Method != "POST" || route != "reading/check" {
		return apperror.NewError(404, "Halaman latihan baca tidak ditemukan.")
	}
	var lessonID, source, transcript string
	var index int
	var audio []byte
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024)
		if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
			return apperror.NewError(400, "Rekaman terlalu panjang.")
		}
		defer r.MultipartForm.RemoveAll()
		lessonID = r.FormValue("lessonId")
		source = r.FormValue("sourceId")
		if json.Unmarshal([]byte(r.FormValue("sentenceIndex")), &index) != nil {
			return apperror.NewError(400, "Kalimat latihan tidak valid.")
		}
		f, _, err := r.FormFile("audio")
		if err != nil {
			return apperror.NewError(400, "Rekam bacaanmu dahulu.")
		}
		defer f.Close()
		audio, err = io.ReadAll(io.LimitReader(f, 1000001))
		if err != nil {
			return err
		}
	} else {
		var in struct {
			LessonID      string `json:"lessonId"`
			SourceID      string `json:"sourceId"`
			SentenceIndex int    `json:"sentenceIndex"`
			Transcript    string `json:"transcript"`
		}
		if err := decode(w, r, &in); err != nil {
			return err
		}
		lessonID, source, index, transcript = in.LessonID, in.SourceID, in.SentenceIndex, strings.TrimSpace(in.Transcript)
	}
	check, err := s.Services.Reading.Check(ctx, uid, model.ReadingRequest{LessonID: lessonID, SourceID: source, SentenceIndex: index, Transcript: transcript, Audio: audio})
	if err != nil {
		return err
	}
	writeJSON(w, 200, map[string]any{"check": check})
	return nil
}
