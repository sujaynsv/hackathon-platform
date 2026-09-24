package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/joho/godotenv"

	"github.com/dogfood-platform/dogfood/internal/config"
	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
)

func main() {
	// 1. Load config from env
	_ = godotenv.Load() // Ignore error if .env doesn't exist (e.g. in prod Docker)

	cfg := config.MustLoad() // panics if required vars are missing

	// STUBS: 2-5
	// db := database.MustConnect(cfg.DatabaseURL)
	// database.MustMigrate(db, "migrations/")
	// rdb := cache.MustConnect(cfg.RedisURL)
	// mc := storage.MustConnect(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey)

	// 6. Wire Chi router
	r := chi.NewRouter()
	
	// Basic middlewares for stub
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{cfg.AllowedOrigins},
	}))
	// r.Use(middleware.Logger())
	// r.Use(middleware.RealIP)
	// r.Use(middleware.Recoverer)

	// 7. Health check (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","version":"0.0.1"}`))
	})

	// 8. Mount module routers (stubs for now)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.JWT(cfg.JWTSecret))
		// auth.Mount(r, authHandler)
		// events.Mount(r, eventsHandler)
		// teams.Mount(r, teamsHandler)
		// submissions.Mount(r, submissionsHandler)
		// judging.Mount(r, judgingHandler)
		// voting.Mount(r, votingHandler)
		// admin.Mount(r, adminHandler)
	})

	// 9. Start server
	addr := ":" + cfg.Port
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
