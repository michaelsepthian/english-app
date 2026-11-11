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

type AudioService struct {
	words    *WordService
	lessons  repository.Lessons
	messages repository.Messages
	limits   *Limiter
}

func (s *AudioService) Text(ctx context.Context, uid string, in model.AudioRequest) (string, error) {
	var err error
	text := ""
	switch in.Kind {
	case "word":
		words, err := s.words.List(ctx, uid)
		if err != nil {
			return "", err
		}
		for _, word := range words {
			if word.ID != in.ID {
				continue
			}
			switch in.Part {
			case "word":
				text = word.Word
			case "v1":
				text = word.V1
			case "v2":
				text = word.V2
			case "v3":
				text = word.V3
			case "example":
				text = word.Example
				if in.ExampleIndex >= 0 && in.ExampleIndex < len(word.Examples) {
					text = word.Examples[in.ExampleIndex].EN
				}
			}
			if strings.ToLower(word.Word) == "read" && in.Part != "example" {
				switch in.Part {
				case "v2":
					text = "Read. I read a book yesterday."
				case "v3":
					text = "Read. I have read this book."
				case "word", "v1":
					text = "Read. I read every day."
				}
			}
		}
	case "listening":
		lesson, err := s.lessons.Find(ctx, uid, in.ID)
		if err != nil {
			return "", err
		}
		if lesson != nil {
			for _, item := range lesson.Content.ListeningItems() {
				if in.SourceID == "" || item.ID == in.SourceID {
					text = item.Transcript
					break
				}
			}
		}
	case "reading":
		lesson, err := s.lessons.Find(ctx, uid, in.ID)
		if err != nil {
			return "", err
		}
		if lesson != nil {
			for _, item := range lesson.Content.ReadingItems() {
				if in.SourceID == "" || item.ID == in.SourceID {
					text = item.Passage
					break
				}
			}
		}
	case "chat":
		messages, err := s.messages.List(ctx, uid)
		if err != nil {
			return "", err
		}
		for _, m := range messages {
			if m.ID == in.ID && m.Role == "assistant" {
				text = m.Content
			}
		}
	}
	if text == "" {
		return "", apperror.NewError(404, "Audio tidak ditemukan.")
	}
	if err = s.limits.Take(ctx, "audio:"+uid, 120, time.Hour); err != nil {
		return "", err
	}
	return text, nil
}
