// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"context"
	"regexp"
	"strings"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/repository"
)

type WordService struct {
	repo   repository.Vocabulary
	tutor  *TutorService
	limits *Limiter
}

var wordPattern = regexp.MustCompile(`^[\pL\pM\s'’-]+$`)

func (s *WordService) List(ctx context.Context, uid string) ([]model.Word, error) {
	words, err := s.repo.List(ctx, uid)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, w := range words {
		seen[strings.ToLower(w.Word)] = true
	}
	for _, w := range model.StarterWords {
		if !seen[strings.ToLower(w.Word)] {
			words = append(words, w)
		}
	}
	return words, nil
}
func (s *WordService) Lookup(ctx context.Context, uid, query string) (model.Word, []model.Word, error) {
	var err error
	query = strings.TrimSpace(query)
	if len(query) < 1 || len(query) > 100 || !wordPattern.MatchString(query) {
		return model.Word{}, nil, apperror.NewError(400, "Masukkan kata atau frasa dalam bahasa Inggris atau Indonesia.")
	}
	ai, err := s.tutor.Client()
	if err != nil {
		return model.Word{}, nil, err
	}
	if err = s.tutor.quota(ctx, uid); err != nil {
		return model.Word{}, nil, err
	}
	release, err := s.limits.Lock(ctx, "word:"+uid)
	if err != nil {
		return model.Word{}, nil, err
	}
	defer release()
	word, err := ai.Lookup(ctx, query)
	if err != nil {
		return model.Word{}, nil, err
	}
	if err = word.Validate(); err != nil {
		return model.Word{}, nil, apperror.NewError(502, err.Error())
	}
	if err = s.repo.Upsert(ctx, uid, word); err != nil {
		return model.Word{}, nil, err
	}
	words, err := s.List(ctx, uid)
	if err != nil {
		return model.Word{}, nil, err
	}
	return word, words, nil
}
