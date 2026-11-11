// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"context"
	"strings"
	"time"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/repository"
)

type ReadingService struct {
	repo    repository.Readings
	lessons repository.Lessons
	tutor   *TutorService
	limits  *Limiter
	speech  Speech
}

func (s *ReadingService) Available() bool { return s.speech != nil }
func (s *ReadingService) List(ctx context.Context, uid, lessonID string) ([]model.ReadingCheck, error) {
	return s.repo.List(ctx, uid, lessonID)
}
func (s *ReadingService) Check(ctx context.Context, uid string, in model.ReadingRequest) (model.ReadingCheck, error) {
	if err := s.limits.Take(ctx, "reading-check:"+uid, 30, time.Hour); err != nil {
		return model.ReadingCheck{}, err
	}
	lessonID, source, index, transcript, audio := in.LessonID, in.SourceID, in.SentenceIndex, strings.TrimSpace(in.Transcript), in.Audio
	if audio == nil && (len(transcript) < 1 || len(transcript) > 3000) {
		return model.ReadingCheck{}, apperror.NewError(400, "Ucapan belum terbaca. Coba lagi.")
	}
	lesson, err := s.lessons.Find(ctx, uid, lessonID)
	if err != nil {
		return model.ReadingCheck{}, err
	}
	if lesson == nil {
		return model.ReadingCheck{}, apperror.NewError(404, "Bacaan tidak ditemukan.")
	}
	reference := ""
	for _, item := range lesson.Content.ReadingItems() {
		if item.ID == source {
			sentences := readingSentences(item.Passage)
			if index >= 0 && index < len(sentences) {
				reference = sentences[index]
			}
		}
	}
	if reference == "" {
		return model.ReadingCheck{}, apperror.NewError(400, "Pilih kalimat yang ingin dilatih.")
	}
	var check model.ReadingCheck
	if audio != nil {
		if s.speech == nil {
			return model.ReadingCheck{}, apperror.NewError(503, "Penilaian bunyi belum tersedia. Kamu bisa memakai latihan kecocokan kata.")
		}
		check, err = s.speech.Assess(ctx, reference, audio)
		if err != nil {
			return model.ReadingCheck{}, err
		}
	} else {
		check = CompareWords(reference, transcript)
	}
	check.SourceID = source
	if ai, e := s.tutor.Client(); e == nil {
		if coach, ok := ai.(interface {
			ReadingFeedback(context.Context, model.ReadingCheck) (string, error)
		}); ok && s.tutor.quota(ctx, uid) == nil {
			if feedback, e := coach.ReadingFeedback(ctx, check); e == nil {
				check.Feedback = feedback
			}
		}
	}
	if err = s.repo.Create(ctx, uid, lessonID, check); err != nil {
		return model.ReadingCheck{}, err
	}
	return check, nil
}
