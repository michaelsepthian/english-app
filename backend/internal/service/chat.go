// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"context"
	"strings"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/repository"
)

type ChatService struct {
	repo   repository.Messages
	tutor  *TutorService
	limits *Limiter
}

func (s *ChatService) List(ctx context.Context, uid string) ([]model.Message, error) {
	return s.repo.List(ctx, uid)
}
func (s *ChatService) Send(ctx context.Context, u model.User, text string) ([]model.Message, error) {
	uid := u.ID
	var err error
	text = strings.TrimSpace(text)
	if len(text) < 1 || len(text) > 4000 {
		return nil, apperror.NewError(400, "Pesan harus berisi 1–2.000 karakter.")
	}
	ai, err := s.tutor.Client()
	if err != nil {
		return nil, err
	}
	release, err := s.limits.Lock(ctx, "chat:"+uid)
	if err != nil {
		return nil, err
	}
	defer release()
	if err = s.tutor.quota(ctx, uid); err != nil {
		return nil, err
	}
	messages, err := s.repo.List(ctx, uid)
	if err != nil {
		return nil, err
	}
	reply, err := ai.Chat(ctx, u.Settings, messages, text)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(reply) == "" {
		return nil, apperror.NewError(502, "Tutor belum bisa menjawab. Coba lagi.")
	}
	if err = s.repo.CreateExchange(ctx, uid, text, reply); err != nil {
		return nil, err
	}
	messages, err = s.repo.List(ctx, uid)
	if err != nil {
		return nil, err
	}
	return messages, nil
}
