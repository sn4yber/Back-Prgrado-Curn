package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sn4yber/curn-networking/internal/adapters/driven/persistence/postgres"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/handler"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/router"
	"github.com/sn4yber/curn-networking/internal/core/usecases/auth"
	"github.com/sn4yber/curn-networking/pkg/config"
	"github.com/sn4yber/curn-networking/pkg/logger"
	"go.uber.org/zap"
)

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.DB)
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
		log,
	)

	// ── 6. Handlers ───────────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authService)

	// ── 7. Router ─────────────────────────────────────────────────────────────
	engine := router.Setup(authHandler, cfg.RateLimit.Requests, cfg.RateLimit.Window, log)

	// ── 8. Servidor HTTP con graceful shutdown ────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      engine,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Arrancamos el servidor en una goroutine para no bloquear el main
	go func() {
		log.Info("servidor iniciado", zap.String("puerto", cfg.App.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("error arrancando servidor", zap.Error(err))
		}
	}()

	// Esperamos señal de apagado (Ctrl+C o SIGTERM del SO)
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
