package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

// ─── Refresh Token ────────────────────────────────────────────────────────────

type refreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *refreshTokenRepository {
	return &refreshTokenRepository{pool: pool}
}

func (r *refreshTokenRepository) Save(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, query, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("refreshTokenRepository.Save: %w", err)
	}
	return nil
}

func (r *refreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `SELECT id, user_id, token_hash, expires_at, created_at FROM refresh_tokens WHERE token_hash = $1 LIMIT 1`
	var t domain.RefreshToken
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("refresh token no encontrado")
		}
		return nil, fmt.Errorf("refreshTokenRepository.FindByTokenHash: %w", err)
	}
	return &t, nil
}

func (r *refreshTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}

func (r *refreshTokenRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

// ─── Password Reset Token ─────────────────────────────────────────────────────

type passwordResetTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPasswordResetTokenRepository(pool *pgxpool.Pool) *passwordResetTokenRepository {
	return &passwordResetTokenRepository{pool: pool}
}

func (r *passwordResetTokenRepository) Save(ctx context.Context, token *domain.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query, token.ID, token.UserID, token.TokenHash, token.ExpiresAt, token.Used, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("passwordResetTokenRepository.Save: %w", err)
	}
	return nil
}

func (r *passwordResetTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	query := `SELECT id, user_id, token_hash, expires_at, used, created_at FROM password_reset_tokens WHERE token_hash = $1 LIMIT 1`
	var t domain.PasswordResetToken
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Used, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("token de recuperación no encontrado")
		}
		return nil, fmt.Errorf("passwordResetTokenRepository.FindByTokenHash: %w", err)
	}
	return &t, nil
}

func (r *passwordResetTokenRepository) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE password_reset_tokens SET used = true WHERE id = $1`, id)
	return err
}

func (r *passwordResetTokenRepository) DeleteExpiredByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM password_reset_tokens WHERE user_id = $1 AND (expires_at < NOW() OR used = true)`, userID)
	return err
}
