// MIVA-PROVIDER: external API-key and speech adapters.
package speech

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/platform/identity"
	"english-app/internal/platform/jsonutil"
)

type SpeechAssessment struct {
	Endpoint, Key string
	HTTP          *http.Client
}

func NewSpeechAssessment(endpoint, key string) (*SpeechAssessment, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !(strings.HasSuffix(u.Hostname(), ".cognitiveservices.azure.com") || strings.HasSuffix(u.Hostname(), ".stt.speech.microsoft.com")) {
		return nil, errors.New("invalid speech endpoint")
	}
	u.Path = "/stt/speech/recognition/conversation/cognitiveservices/v1"
	if strings.HasSuffix(u.Hostname(), ".stt.speech.microsoft.com") {
		u.Path = "/speech/recognition/conversation/cognitiveservices/v1"
	}
	u.RawQuery = "language=en-US&format=detailed"
	return &SpeechAssessment{
		u.String(),
		key,
		&http.Client{
			Timeout:       45 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func ValidateWav(data []byte) error {
	if len(data) < 44 || len(data) > 1000000 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return apperror.NewError(400, "Rekaman belum valid. Coba rekam ulang maksimal 30 detik.")
	}
	fmtOK, dataOK := false, false
	for offset := 12; offset+8 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[offset+4:]))
		start := offset + 8
		if size < 0 || size > len(data)-start {
			return apperror.NewError(400, "Data rekaman belum lengkap.")
		}
		switch string(data[offset : offset+4]) {
		case "fmt ":
			if size < 16 || binary.LittleEndian.Uint16(data[start:]) != 1 || binary.LittleEndian.Uint16(data[start+2:]) != 1 || binary.LittleEndian.Uint32(data[start+4:]) != 16000 || binary.LittleEndian.Uint16(data[start+14:]) != 16 {
				return apperror.NewError(400, "Format rekaman belum sesuai.")
			}
			fmtOK = true
		case "data":
			if size < 32000 || size > 960000 {
				return apperror.NewError(400, "Rekam antara 1 dan 30 detik.")
			}
			dataOK = true
		}
		offset = start + size + (size % 2)
	}
	if !fmtOK || !dataOK {
		return apperror.NewError(400, "Rekaman belum dapat dibaca.")
	}
	return nil
}

func (s *SpeechAssessment) Assess(ctx context.Context, reference string, audio []byte) (model.ReadingCheck, error) {
	var out model.ReadingCheck
	if err := ValidateWav(audio); err != nil {
		return out, err
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", s.Endpoint, bytes.NewReader(audio))
	req.Header.Set("Ocp-Apim-Subscription-Key", s.Key)
	req.Header.Set("Content-Type", "audio/wav; codecs=audio/pcm; samplerate=16000")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Pronunciation-Assessment", base64.StdEncoding.EncodeToString(jsonutil.Marshal(map[string]any{
		"ReferenceText": reference,
		"GradingSystem": "HundredMark",
		"Granularity":   "Word",
		"Dimension":     "Comprehensive",
		"EnableMiscue":  true,
	})))
	response, err := s.HTTP.Do(req)
	if err != nil {
		return out, apperror.NewError(502, "Rekaman belum dapat diperiksa. Coba lagi.")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return out, apperror.NewError(502, "Pemeriksaan pelafalan belum tersedia saat ini.")
	}
	var result struct {
		RecognitionStatus string
		NBest             []struct {
			Display                                                   string
			AccuracyScore, FluencyScore, CompletenessScore, PronScore *float64
			PronunciationAssessment                                   *struct{ AccuracyScore, FluencyScore, CompletenessScore, PronScore *float64 }
			Words                                                     []struct {
				Word, ErrorType         string
				AccuracyScore           *float64
				PronunciationAssessment *struct {
					AccuracyScore *float64
					ErrorType     string
				}
			}
		}
	}
	if json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&result) != nil || result.RecognitionStatus != "Success" || len(result.NBest) == 0 {
		return out, apperror.NewError(422, "Suara belum terbaca dengan jelas. Coba rekam di tempat yang lebih tenang.")
	}
	best := result.NBest[0]
	accuracy, fluency, completeness, score := best.AccuracyScore, best.FluencyScore, best.CompletenessScore, best.PronScore
	if best.PronunciationAssessment != nil {
		accuracy = best.PronunciationAssessment.AccuracyScore
		fluency = best.PronunciationAssessment.FluencyScore
		completeness = best.PronunciationAssessment.CompletenessScore
		score = best.PronunciationAssessment.PronScore
	}
	if score == nil || accuracy == nil || fluency == nil || completeness == nil {
		return out, apperror.NewError(502, "Skor pelafalan belum tersedia untuk rekaman ini. Tidak ada skor yang dibuat-buat.")
	}
	for _, n := range []*float64{score, accuracy, fluency, completeness} {
		if math.IsNaN(*n) || *n < 0 || *n > 100 {
			return out, apperror.NewError(502, "Hasil pelafalan belum valid.")
		}
	}
	out = model.ReadingCheck{
		ID:           identity.NewID(),
		Mode:         "pronunciation",
		Label:        "Pelafalan",
		Reference:    reference,
		Transcript:   best.Display,
		Score:        *score,
		Accuracy:     *accuracy,
		Fluency:      *fluency,
		Completeness: *completeness,
		Words:        []model.ReadingWord{},
		Feedback:     "Skor berasal dari rekaman suara. Perhatikan kata yang perlu diulang; aksen tidak perlu sempurna agar ucapanmu dapat dipahami.",
	}
	for _, w := range best.Words {
		status, acc := w.ErrorType, w.AccuracyScore
		if w.PronunciationAssessment != nil {
			status = w.PronunciationAssessment.ErrorType
			acc = w.PronunciationAssessment.AccuracyScore
		}
		if status == "None" {
			status = "correct"
		}
		item := model.ReadingWord{Word: w.Word, Status: status}
		if acc != nil {
			item.Accuracy = *acc
		}
		out.Words = append(out.Words, item)
	}
	return out, nil
}
