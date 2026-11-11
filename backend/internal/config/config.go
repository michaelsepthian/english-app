// MIVA-CONFIG: environment configuration.
package config

import (
	"net/url"
	"os"
	"strconv"

	"english-app/internal/handler"
	"english-app/internal/service"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type Settings struct {
	DatabaseURL                              string
	MigrateOnStart                           bool
	APIKey, BaseURL, Model, Format, JSONMode string
	SpeechKey, SpeechEndpoint                string
	Address                                  string
	HTTP                                     handler.Config
	Service                                  service.Config
}

func Load() Settings {
	databaseURL := url.URL{Scheme: "postgres", User: url.UserPassword(env("DB_USER", "english_app"), os.Getenv("DB_PASSWORD")), Host: env("DB_HOST", "localhost") + ":" + env("DB_PORT", "55432"), Path: env("DB_NAME", "toko")}
	q := databaseURL.Query()
	q.Set("sslmode", env("DB_SSLMODE", "disable"))
	databaseURL.RawQuery = q.Encode()
	limit, _ := strconv.Atoi(env("AI_DAILY_LIMIT", "120"))
	if limit < 1 {
		limit = 120
	}
	return Settings{
		DatabaseURL: databaseURL.String(), MigrateOnStart: env("MIGRATE_ON_START", "true") == "true",
		APIKey: os.Getenv("AI_API_KEY"), BaseURL: env("AI_BASE_URL", "https://member.vpsindonesia.co.id/api/v1"), Model: env("AI_MODEL", "sonnet-5"), Format: env("AI_API_FORMAT", "chat_completions"), JSONMode: env("AI_JSON_MODE", "prompt"),
		SpeechKey: os.Getenv("AZURE_SPEECH_KEY"), SpeechEndpoint: os.Getenv("AZURE_SPEECH_ENDPOINT"), Address: env("HTTP_ADDR", ":8080"),
		HTTP:    handler.Config{Origin: env("APP_ORIGIN", "http://localhost:5180"), StaticDir: env("STATIC_DIR", "../frontend/dist"), SecureCookie: env("COOKIE_SECURE", "false") == "true"},
		Service: service.Config{AIConfigured: os.Getenv("AI_API_KEY") != "", Registration: env("ALLOW_REGISTRATION", "true") == "true", DailyLimit: limit, APIModel: env("AI_MODEL", "sonnet-5")},
	}
}
