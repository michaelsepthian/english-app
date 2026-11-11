// Package service owns application rules. Dependencies are interfaces; no SQL or HTTP transport belongs here.
package service

import (
	"context"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
)

type TutorService struct {
	ai         AI
	configured bool
	limits     *Limiter
	dailyLimit int
	model      string
}

func (t *TutorService) Client() (AI, error) {
	if !t.configured || t.ai == nil {
		return nil, apperror.NewError(503, "API key belum diisi. Isi AI_API_KEY di backend/.env, lalu jalankan ulang aplikasi.")
	}
	return t.ai, nil
}
func (t *TutorService) Configured() bool { return t.configured && t.ai != nil }
func (t *TutorService) quota(ctx context.Context, uid string) error {
	return t.limits.Take(ctx, "ai:"+uid+":"+model.DayKey(time.Now()), t.dailyLimit, 24*time.Hour)
}
func (t *TutorService) Probe(ctx context.Context, uid string) (map[string]any, error) {
	ai, err := t.Client()
	if err != nil {
		return nil, err
	}
	if err = t.quota(ctx, uid); err != nil {
		return nil, err
	}
	if err = t.limits.Take(ctx, "api-test:"+uid, 3, 10*time.Minute); err != nil {
		return nil, err
	}
	tester, ok := ai.(interface {
		Probe(context.Context) (string, error)
	})
	if !ok {
		return nil, apperror.NewError(500, "Penyedia AI belum mendukung tes koneksi.")
	}
	if _, err = tester.Probe(ctx); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "message": "Koneksi API berhasil. Tutor siap digunakan.", "model": t.model}, nil
}
