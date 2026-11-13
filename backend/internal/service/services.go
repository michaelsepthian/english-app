// Package service owns application rules. Dependencies are interfaces; no SQL or HTTP transport belongs here.
package service

import (
	"context"
	"english-app/internal/repository"
)

type Config struct {
	AIConfigured, Registration bool
	DailyLimit                 int
	APIModel                   string
}
type Services struct {
	Auth     *AuthService
	Tutor    *TutorService
	Words    *WordService
	Chat     *ChatService
	Lessons  *LessonService
	Progress *ProgressService
	Studies  *StudyService
	Reviews  *ReviewService
	Reading  *ReadingService
	Audio    *AudioService
	Profile  *ProfileService
	Health   repository.Health
}

func New(r repository.Repositories, ai AI, speech Speech, c Config) *Services {
	if c.DailyLimit < 1 {
		c.DailyLimit = 120
	}
	workerContext, cancelWorkers := context.WithCancel(context.Background())
	limits := &Limiter{repo: r.Limits}
	tutor := &TutorService{ai: ai, configured: c.AIConfigured, limits: limits, dailyLimit: c.DailyLimit, model: c.APIModel}
	reviews := &ReviewService{lessons: r.Lessons, users: r.Users, tutor: tutor}
	words := &WordService{repo: r.Words, tutor: tutor, limits: limits}
	lessons := &LessonService{repo: r.Lessons, tutor: tutor, limits: limits}
	return &Services{
		Auth: &AuthService{users: r.Users, sessions: r.Sessions, limits: limits, registration: c.Registration}, Tutor: tutor,
		Words: words, Chat: &ChatService{repo: r.Messages, tutor: tutor, limits: limits}, Lessons: lessons,
		Progress: &ProgressService{repo: r.Lessons}, Reviews: reviews,
		Studies: &StudyService{jobs: r.Jobs, users: r.Users, lessons: r.Lessons, tutor: tutor, limits: limits, workers: make(chan struct{}, 2), workerContext: workerContext, cancelWorkers: cancelWorkers},
		Reading: &ReadingService{repo: r.Readings, lessons: r.Lessons, tutor: tutor, limits: limits, speech: speech},
		Audio:   &AudioService{words: words, lessons: r.Lessons, messages: r.Messages, limits: limits}, Profile: &ProfileService{users: r.Users}, Health: r.Health,
	}
}
