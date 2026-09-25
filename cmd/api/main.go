package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/joho/godotenv"

	"github.com/dogfood-platform/dogfood/internal/auth/handler"
	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/auth/repository"
	"github.com/dogfood-platform/dogfood/internal/auth/usecase"
	"github.com/dogfood-platform/dogfood/internal/config"
	"github.com/dogfood-platform/dogfood/internal/shared"
	"github.com/dogfood-platform/dogfood/internal/shared/cache"
	"github.com/dogfood-platform/dogfood/internal/shared/database"
	"github.com/dogfood-platform/dogfood/internal/shared/email"
	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	"github.com/dogfood-platform/dogfood/internal/shared/captcha"
)

func main() {
	// 1. Load config from env
	_ = godotenv.Load() // Ignore error if .env doesn't exist (e.g. in prod Docker)

	cfg := config.MustLoad() // panics if required vars are missing

	// 2. Connect to Database and Migrate
	db := database.MustConnect(cfg.DatabaseURL)
	database.MustMigrate(db, "migrations/")
	
	// 3. Connect to Cache
	rdb := cache.MustConnect(cfg.RedisURL)
	rateLimiter := middleware.NewRateLimiter(rdb)

	// STUBS: 4-5
	// mc := storage.MustConnect(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey)
	
	// Initialize Auth dependencies
	userRepo := repository.NewUserRepository(db)
	hasher := shared.NewBcryptHasher()
	tokenIssuer := shared.NewJWTIssuer(cfg.JWTSecret, cfg.JWTAccessTTLH*60, cfg.JWTRefreshTTLD)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	passwordValidator := shared.NewHIBPValidator()
	emailTokensRepo := repository.NewEmailVerificationRepository(db)
	emailSender := email.NewStubSender(slog.Default())
	
	var captchaValidator port.CaptchaValidator
	if cfg.TurnstileKey != "" {
		captchaValidator = captcha.NewTurnstileValidator(cfg.TurnstileKey)
	}

	registerSvc := usecase.NewRegisterService(userRepo, hasher, tokenIssuer, refreshRepo, passwordValidator, emailTokensRepo, emailSender, captchaValidator)
	verifySvc := usecase.NewVerifyEmailService(emailTokensRepo, userRepo)
	
	webAuthnRepo := repository.NewWebAuthnRepository(db)
	redisCache := cache.NewRedisCache(rdb)
	webAuthnSvc, err := usecase.NewWebAuthnService(
		userRepo, webAuthnRepo, redisCache, tokenIssuer, refreshRepo, 
		"Dogfood Hackathon", "localhost", "http://localhost:3000",
	)
	if err != nil {
		slog.Error("failed to init webauthn", "error", err)
		os.Exit(1)
	}

	authHandler := handler.NewAuthHandler(registerSvc, verifySvc)
	webAuthnHandler := handler.NewWebAuthnHandler(webAuthnSvc)

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
		// Public routes
		r.Group(func(r chi.Router) {
			// Rate limit: 5 requests per 15 minutes per IP
			r.Use(rateLimiter.RateLimit(5, 15*time.Minute))
			r.Mount("/auth", authHandler.Routes())
			webAuthnHandler.RegisterRoutes(r)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWT(cfg.JWTSecret))
			// events.Mount(r, eventsHandler)
			// teams.Mount(r, teamsHandler)
			// submissions.Mount(r, submissionsHandler)
			// judging.Mount(r, judgingHandler)
			// voting.Mount(r, votingHandler)
			// admin.Mount(r, adminHandler)
		})
	})

	// 9. Start server
	addr := ":" + cfg.Port
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
