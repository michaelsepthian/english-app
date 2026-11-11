// Package service owns application rules. Dependencies are interfaces; no SQL or HTTP transport belongs here.
package service

import (
	"context"
	"time"

	"english-app/internal/platform/apperror"
	"english-app/internal/repository"
)

type Limiter struct{ repo repository.Limits }

func (l *Limiter) Take(ctx context.Context, key string, max int, window time.Duration) error {
	n, err := l.repo.Increment(ctx, key, time.Now().Add(window))
	if err != nil {
		return err
	}
	if n > max {
		return apperror.NewError(429, "Batas permintaan tercapai. Coba lagi nanti.")
	}
	return nil
}
func (l *Limiter) Lock(ctx context.Context, key string) (func(), error) {
	ok, err := l.repo.Acquire(ctx, key, time.Now().Add(2*time.Minute))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.NewError(409, "Permintaan sebelumnya masih diproses. Tunggu sebentar.")
	}
	return func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = l.repo.Release(c, key)
	}, nil
}
