package main

import (
	"context"
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
	"github.com/dogfood-platform/dogfood/internal/shared/captcha"
	"github.com/dogfood-platform/dogfood/internal/shared/database"
	"github.com/dogfood-platform/dogfood/internal/shared/email"
	"github.com/dogfood-platform/dogfood/internal/shared/middleware"
	"github.com/dogfood-platform/dogfood/internal/shared/storage"

	eventsHandlerPkg "github.com/dogfood-platform/dogfood/internal/events/handler"
	eventRepo "github.com/dogfood-platform/dogfood/internal/events/repository"
	eventsUsecase "github.com/dogfood-platform/dogfood/internal/events/usecase"
	subHandlerPkg "github.com/dogfood-platform/dogfood/internal/submissions/handler"
	subRepo "github.com/dogfood-platform/dogfood/internal/submissions/repository"
	subUsecase "github.com/dogfood-platform/dogfood/internal/submissions/usecase"
	teamsHandlerPkg "github.com/dogfood-platform/dogfood/internal/teams/handler"
	teamRepo "github.com/dogfood-platform/dogfood/internal/teams/repository"
	teamsUsecase "github.com/dogfood-platform/dogfood/internal/teams/usecase"
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
	minioStorage, err := storage.NewMinIOStorage(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioUseSSL)
	if err != nil {
		slog.Error("failed to connect to minio", "err", err)
		panic(err)
	}

	// Initialize Auth dependencies
	userRepo := repository.NewUserRepository(db)
	hasher := shared.NewBcryptHasher()
	tokenIssuer := shared.NewJWTIssuer(cfg.JWTSecret, cfg.JWTAccessTTLH*60, cfg.JWTRefreshTTLD)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	passwordValidator := shared.NewHIBPValidator()
	emailTokensRepo := repository.NewEmailVerificationRepository(db)
	var baseEmailSender port.EmailSender
	if cfg.SMTPHost != "" {
		baseEmailSender = email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	} else {
		baseEmailSender = email.NewStubSender(slog.Default())
	}

	// Wrap the base sender in the queue sender for async delivery
	emailQueueSender := email.NewQueueSender(rdb)

	// Start the background worker
	go email.RunWorker(context.Background(), rdb, baseEmailSender)

	var captchaValidator port.CaptchaValidator
	if cfg.TurnstileKey != "" {
		captchaValidator = captcha.NewTurnstileValidator(cfg.TurnstileKey)
	}

	registerSvc := usecase.NewRegisterService(userRepo, hasher, tokenIssuer, refreshRepo, passwordValidator, emailTokensRepo, emailQueueSender, captchaValidator)
	verifySvc := usecase.NewVerifyEmailService(emailTokensRepo, userRepo)
	loginSvc := usecase.NewLoginService(userRepo, hasher, tokenIssuer, refreshRepo)
	refreshSvc := usecase.NewRefreshService(userRepo, refreshRepo, tokenIssuer)
	redisCache := cache.NewRedisCache(rdb)
	logoutSvc := usecase.NewLogoutService(redisCache, refreshRepo)

	webAuthnRepo := repository.NewWebAuthnRepository(db)
	webAuthnSvc, err := usecase.NewWebAuthnService(
		userRepo, webAuthnRepo, redisCache, tokenIssuer, refreshRepo,
		"Dogfood Hackathon", cfg.WebAuthnRPID, cfg.WebAuthnRPOrigin,
	)
	if err != nil {
		slog.Error("failed to init webauthn", "error", err)
		os.Exit(1)
	}

	authHandler := handler.NewAuthHandler(registerSvc, verifySvc, loginSvc, refreshSvc, logoutSvc, redisCache, cfg.JWTSecret)
	webAuthnHandler := handler.NewWebAuthnHandler(webAuthnSvc)

	eventsRepo := eventRepo.NewPgEventRepository(db)
	eventsRoleRepo := eventRepo.NewPgEventRoleRepository(db)
	eventsTrackRepo := eventRepo.NewPgTrackRepository(db)
	eventsCreateSvc := eventsUsecase.NewCreateEventService(eventsRepo, eventsRoleRepo)
	eventsListSvc := eventsUsecase.NewListEventsService(eventsRepo)
	eventsGetSvc := eventsUsecase.NewGetEventService(eventsRepo)
	eventsUpdateSvc := eventsUsecase.NewUpdateEventService(eventsRepo, eventsRoleRepo, eventsTrackRepo)
	eventsHandler := eventsHandlerPkg.NewEventHandler(eventsCreateSvc, eventsListSvc, eventsGetSvc, eventsUpdateSvc)

	// Submissions module
	subEventsReader := eventRepo.NewPgEventReader(db)
	teamsRepo := teamRepo.NewPgTeamReader(db)
	tracksRepo := eventRepo.NewPgTrackReader(db)
	subsRepo := subRepo.NewPgSubmissionRepository(db)
	uploadRepo := subRepo.NewPgUploadRepository(db)

	createSubSvc := subUsecase.NewCreateSubmissionService(subEventsReader, teamsRepo, tracksRepo, subsRepo)
	updateSubSvc := subUsecase.NewUpdateSubmissionService(subEventsReader, teamsRepo, tracksRepo, subsRepo)
	submitSubSvc := subUsecase.NewFinalSubmitService(subEventsReader, teamsRepo, tracksRepo, subsRepo)
	uploadSubSvc := subUsecase.NewUploadService(subEventsReader, teamsRepo, subsRepo, uploadRepo, minioStorage)

	subHandler := subHandlerPkg.NewSubmissionHandler(createSubSvc, updateSubSvc, submitSubSvc, uploadSubSvc, uploadSubSvc)

	// Event registration and unregistration
	registrationEvents := teamRepo.NewPgEventReader(db)
	participantsRepo := teamRepo.NewPgParticipantRepository(db)
	registerForEventSvc := teamsUsecase.NewRegisterService(registrationEvents, participantsRepo)
	unregisterSvc := teamsUsecase.NewUnregisterService(registrationEvents, participantsRepo, teamsRepo)
	teamsHandler := teamsHandlerPkg.NewTeamsHandler(registerForEventSvc, unregisterSvc)

	// 6. Wire Chi router
	r := chi.NewRouter()

	// Basic middlewares for stub
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.AllowedOrigins},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
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
			webAuthnHandler.RegisterPublicRoutes(r)
		})

		r.Group(func(r chi.Router) {
			// Public reads are used by server-rendered pages and have a separate, higher limit.
			r.Use(middleware.OptionalJWTMiddleware(cfg.JWTSecret, redisCache))
			r.Use(rateLimiter.RateLimit(120, time.Minute))
			eventsHandler.RegisterPublicRoutes(r)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTMiddleware(cfg.JWTSecret, redisCache))
			webAuthnHandler.RegisterProtectedRoutes(r)
			subHandler.RegisterRoutes(r)
			eventsHandler.RegisterProtectedRoutes(r)
			teamsHandler.RegisterRoutes(r)
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
