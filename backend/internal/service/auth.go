// MIVA-SERVICE: learning workflows, account rules and progress.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"english-app/internal/model"
	"english-app/internal/platform/apperror"
	"english-app/internal/platform/identity"
	"english-app/internal/repository"
)

type AuthService struct {
	users        repository.Users
	sessions     repository.Sessions
	limits       *Limiter
	registration bool
}

func (s *AuthService) Authenticate(ctx context.Context, in model.Credentials, ip string, register bool) (model.User, string, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	email, err := mail.ParseAddress(in.Email)
	if err != nil || email.Address != in.Email || len(in.Email) > 254 || len(in.Password) < 10 || len(in.Password) > 72 {
		return model.User{}, "", apperror.NewError(400, "Gunakan email yang valid dan kata sandi 10–72 byte.")
	}
	if err = s.limits.Take(ctx, "login:"+identity.HashToken(in.Email), 10, 15*time.Minute); err != nil {
		return model.User{}, "", err
	}
	if err = s.limits.Take(ctx, "auth-ip:"+identity.HashToken(ip), 40, 15*time.Minute); err != nil {
		return model.User{}, "", err
	}
	u, password, err := s.users.FindByEmail(ctx, in.Email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return u, "", err
	}
	if register {
		if !s.registration {
			return u, "", apperror.NewError(403, "Pendaftaran akun baru sedang ditutup.")
		}
		if utf8.RuneCountInString(in.Name) < 2 || len(in.Name) > 120 {
			return u, "", apperror.NewError(400, "Nama harus terdiri dari 2–60 karakter.")
		}
		if err == nil {
			return u, "", apperror.NewError(409, "Email sudah terdaftar. Silakan masuk.")
		}
		hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
		if e != nil {
			return u, "", e
		}
		u = model.User{ID: identity.NewID(), Email: in.Email, Settings: model.DefaultSettings(in.Name)}
		if e = s.users.Create(ctx, u, string(hash)); e != nil {
			if errors.Is(e, repository.ErrConflict) {
				e = apperror.NewError(409, "Email sudah terdaftar. Silakan masuk.")
			}
			return u, "", e
		}
	} else {
		if errors.Is(err, repository.ErrNotFound) {
			password = "$2a$12$j9JfB5j/p44a4mYG7i6RWu4y8KvxRlEo7BEZb/PY1vp92z0.qfO8e"
		}
		if bcrypt.CompareHashAndPassword([]byte(password), []byte(in.Password)) != nil || u.ID == "" {
			return model.User{}, "", apperror.NewError(401, "Email atau kata sandi tidak cocok.")
		}
	}
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return u, "", err
	}
	token := hex.EncodeToString(bytes)
	err = s.sessions.Create(ctx, u.ID, identity.HashToken(token), time.Now().Add(7*24*time.Hour))
	return u, token, err
}
func (s *AuthService) Session(ctx context.Context, token string) (model.User, error) {
	if len(token) != 64 {
		return model.User{}, apperror.NewError(401, "Silakan masuk terlebih dahulu.")
	}
	id, err := s.sessions.FindUserID(ctx, identity.HashToken(token))
	if errors.Is(err, repository.ErrNotFound) {
		return model.User{}, apperror.NewError(401, "Sesi berakhir. Silakan masuk lagi.")
	}
	if err != nil {
		return model.User{}, err
	}
	u, err := s.users.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.User{}, apperror.NewError(401, "Sesi berakhir. Silakan masuk lagi.")
	}
	return u, err
}
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, identity.HashToken(token))
}

type ProfileService struct{ users repository.Users }

func (s *ProfileService) Update(ctx context.Context, u model.User, settings model.Settings) (model.User, error) {
	if err := settings.Validate(); err != nil {
		return u, apperror.NewError(400, err.Error())
	}
	if err := s.users.UpdateSettings(ctx, u.ID, settings); err != nil {
		return u, err
	}
	u.Settings = settings
	return u, nil
}
