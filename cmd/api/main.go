package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/adapters/driven/persistence/postgres"
	"github.com/sn4yber/curn-networking/internal/adapters/driven/storage"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/handler"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/router"
	"github.com/sn4yber/curn-networking/internal/core/usecases/auth"
	"github.com/sn4yber/curn-networking/internal/core/usecases/comment"
	"github.com/sn4yber/curn-networking/internal/core/usecases/connection"
	"github.com/sn4yber/curn-networking/internal/core/usecases/conversation"
	"github.com/sn4yber/curn-networking/internal/core/usecases/mentorship"
	"github.com/sn4yber/curn-networking/internal/core/usecases/moderation"
	"github.com/sn4yber/curn-networking/internal/core/usecases/notification"
	"github.com/sn4yber/curn-networking/internal/core/usecases/points"
	"github.com/sn4yber/curn-networking/internal/core/usecases/post"
	"github.com/sn4yber/curn-networking/internal/core/usecases/project"
	"github.com/sn4yber/curn-networking/internal/core/usecases/user"
	"github.com/sn4yber/curn-networking/pkg/config"
	"github.com/sn4yber/curn-networking/pkg/logger"
	"go.uber.org/zap"
)

const (
	dbConnectMaxAttempts = 10
	dbConnectRetryDelay  = 3 * time.Second
	dbConnectTryTimeout  = 8 * time.Second
)

func connectDBWithRetry(cfg config.DBConfig, log logger.Logger) (*pgxpool.Pool, error) {
	var lastErr error

	for attempt := 1; attempt <= dbConnectMaxAttempts; attempt++ {
		attemptCtx, cancel := context.WithTimeout(context.Background(), dbConnectTryTimeout)
		pool, err := postgres.NewPool(attemptCtx, cfg)
		cancel()

		if err == nil {
			if attempt > 1 {
				log.Info("conexión a PostgreSQL recuperada tras reintentos", zap.Int("intentos", attempt))
			}
			return pool, nil
		}

		lastErr = err
		log.Warn("PostgreSQL aún no está listo; reintentando conexión",
			zap.Int("intento", attempt),
			zap.Int("max_intentos", dbConnectMaxAttempts),
			zap.Error(err),
		)

		if attempt < dbConnectMaxAttempts {
			time.Sleep(dbConnectRetryDelay)
		}
	}

	return nil, fmt.Errorf("no se pudo conectar a PostgreSQL tras %d intentos: %w", dbConnectMaxAttempts, lastErr)
}

func main() {
	// ── 1. Configuración ──────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		panic("error cargando configuración: " + err.Error())
	}

	// ── 2. Logger ─────────────────────────────────────────────────────────────
	log, err := logger.New(cfg.App.Env)
	if err != nil {
		panic("error inicializando logger: " + err.Error())
	}

	// ── 3. Base de datos ──────────────────────────────────────────────────────
	pool, err := connectDBWithRetry(cfg.DB, log)
	if err != nil {
		log.Fatal("error conectando a PostgreSQL", zap.Error(err))
	}
	defer pool.Close()

	log.Info("conexión a PostgreSQL establecida",
		zap.String("host", cfg.DB.Host),
		zap.String("db", cfg.DB.Name),
	)

	// ── 4. Repositorios ───────────────────────────────────────────────────────
	userRepo := postgres.NewUserRepository(pool)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(pool)
	resetTokenRepo := postgres.NewPasswordResetTokenRepository(pool)
	connectionRepo := postgres.NewConnectionRepositoryPostgres(pool)
	conversationRepo := postgres.NewConversationRepository(pool)
	postRepo := postgres.NewPostRepository(pool)
	commentRepo := postgres.NewCommentRepository(pool)
	projectRepo := postgres.NewProjectRepository(pool)
	mentorshipRepo := postgres.NewMentorshipRepository(pool)
	notificationRepo := postgres.NewNotificationRepository(pool)
	fileStorageURL := os.Getenv("FILE_STORAGE_BASE_URL")
	if fileStorageURL == "" {
		fileStorageURL = "http://localhost:" + cfg.App.Port + "/uploads"
	}
	fileStorage := storage.NewLocalFileStorage("./uploads", fileStorageURL)

	// ── 5. Casos de uso ───────────────────────────────────────────────────────
	authService := auth.New(
		userRepo,
		refreshTokenRepo,
		resetTokenRepo,
		cfg.Argon2.Memory,
		cfg.Argon2.Iterations,
		cfg.Argon2.Parallelism,
		cfg.Argon2.KeyLength,
		cfg.JWT.Secret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
		cfg.Auth.DefaultProgramID,
		log,
	)
	connectionUseCase := connection.NewConnectionUseCase(connectionRepo)
	userService := user.New(userRepo, connectionRepo, fileStorage, log)
	conversationService := conversation.New(conversationRepo, userRepo, postRepo, fileStorage)
	postService := post.New(postRepo, userRepo, fileStorage)
	commentService := comment.New(commentRepo, userRepo, postRepo)
	moderationService := moderation.New(postRepo, userRepo)
	projectService := project.New(projectRepo, userRepo)
	notificationService := notification.New(notificationRepo)
	mentorshipService := mentorship.New(mentorshipRepo, userRepo, projectRepo, notificationRepo)
	pointsRepo := postgres.NewPointsRepository(pool)
	pointsService := points.New(pointsRepo)

	// ── 6. Handlers ───────────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authService)
	connectionHandler := handler.NewConnectionHandler(connectionUseCase)
	userHandler := handler.NewUserHandler(userService)
	conversationHandler := handler.NewConversationHandler(conversationService)
	postHandler := handler.NewPostHandler(postService)
	commentHandler := handler.NewCommentHandler(commentService)
	moderationHandler := handler.NewModerationHandler(moderationService)
	projectHandler := handler.NewProjectHandler(projectService)
	mentorshipHandler := handler.NewMentorshipHandler(mentorshipService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	pointsHandler := handler.NewPointsHandler(pointsService)

	// ── 7. Router ─────────────────────────────────────────────────────────────
	engine := router.Setup(
		authHandler,
		connectionHandler,
		userHandler,
		conversationHandler,
		postHandler,
		moderationHandler,
		projectHandler,
		mentorshipHandler,
		notificationHandler,
		commentHandler,
		pointsHandler,
		cfg.JWT.Secret,
		cfg.CORS.AllowedOrigins,
		cfg.RateLimit.Requests,
		cfg.RateLimit.Window,
		log,
		pool, // Pool para health check
	)

	// ── 8. Servidor HTTP con graceful shutdown ────────────────────────────────
	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: engine,
		// ReadTimeout: tiempo máximo para leer el request completo (headers + body)
		ReadTimeout: 15 * time.Second,
		// WriteTimeout: tiempo máximo para escribir la respuesta completa
		WriteTimeout: 30 * time.Second,
		// IdleTimeout: tiempo máximo que una conexión keep-alive puede estar inactiva
		IdleTimeout: 120 * time.Second,
		// ReadHeaderTimeout: protección específica contra Slowloris attacks
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("servidor iniciado", zap.String("puerto", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("error arrancando servidor", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("apagando servidor...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("error durante el apagado", zap.Error(err))
	}

	log.Info("servidor detenido correctamente")
}
