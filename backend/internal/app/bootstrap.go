// MIVA-APP: dependency composition and application lifecycle.
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"english-app/internal/config"
	"english-app/internal/database"
	"english-app/internal/handler"
	"english-app/internal/provider/ai"
	"english-app/internal/provider/speech"
	"english-app/internal/repository/postgres"
	"english-app/internal/service"
)

func Run() {
	_ = godotenv.Load()
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if len(os.Args) > 2 || (command != "serve" && command != "migrate") {
		slog.Error("Gunakan: english-app [serve|migrate]")
		os.Exit(2)
	}
	settings := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	pool, err := database.Open(ctx, settings.DatabaseURL)
	cancel()
	if err != nil {
		slog.Error("PostgreSQL belum siap. Periksa DB_* dan hak akses schema english_app.")
		os.Exit(1)
	}
	defer pool.Close()
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
	if command == "migrate" || settings.MigrateOnStart {
		err = database.Migrate(ctx, pool)
	} else {
		err = database.CheckMigrations(ctx, pool)
	}
	cancel()
	if err != nil {
		slog.Error("Migrasi database belum sesuai; deployment dihentikan.", "error", err)
		os.Exit(1)
	}
	if command == "migrate" {
		slog.Info("Migrasi database selesai")
		return
	}

	cfg := settings.Service
	apiClient := ai.NewAI(settings.APIKey, settings.Model)
	if e := apiClient.ConfigureCompatible(settings.BaseURL, settings.Format, settings.JSONMode); e != nil {
		slog.Error("Konfigurasi API belum valid. Periksa AI_BASE_URL, AI_API_FORMAT, AI_JSON_MODE.")
		os.Exit(1)
	}

	var assessment service.Speech
	if settings.SpeechKey != "" {
		client, e := speech.NewSpeechAssessment(settings.SpeechEndpoint, settings.SpeechKey)
		if e != nil {
			slog.Error("Konfigurasi pemeriksaan suara belum valid.")
			os.Exit(1)
		}
		assessment = client
	}
	repos := postgres.New(pool)
	recoveryCtx, recoveryCancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = repos.Jobs.RecoverInterrupted(recoveryCtx)
	recoveryCancel()
	if err != nil {
		slog.Error("Pemulihan sesi belum berhasil.")
		os.Exit(1)
	}
	services := service.New(repos, apiClient, assessment, cfg)
	server := &handler.Server{Services: services, Config: settings.HTTP}
	httpServer := &http.Server{
		Addr:              settings.Address,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      110 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}
	go func() {
		slog.Info("Miva siap", "address", httpServer.Addr, "AI", cfg.AIConfigured)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server gagal dijalankan")
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
	if err := services.Studies.Shutdown(ctx); err != nil {
		slog.Warn("Worker belum berhenti sebelum shutdown")
	}
}
