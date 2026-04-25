package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type pointsRepository struct {
	pool *pgxpool.Pool
}

func NewPointsRepository(pool *pgxpool.Pool) *pointsRepository {
	return &pointsRepository{pool: pool}
}

func (r *pointsRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserPoints, error) {
	var p domain.UserPoints
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, points, streak, last_active, created_at, updated_at
		FROM user_points
		WHERE user_id = $1
	`, userID).Scan(&p.UserID, &p.Points, &p.Streak, &p.LastActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("pointsRepository.GetByUserID: %w", err)
	}
	return &p, nil
}

func (r *pointsRepository) Upsert(ctx context.Context, points *domain.UserPoints) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_points (user_id, points, streak, last_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			points = $2,
			streak = $3,
			last_active = $4,
			updated_at = $6
	`, points.UserID, points.Points, points.Streak, points.LastActive, points.CreatedAt, points.UpdatedAt)
	if err != nil {
		return fmt.Errorf("pointsRepository.Upsert: %w", err)
	}
	return nil
}

func (r *pointsRepository) ListAchievements(ctx context.Context, userID uuid.UUID) ([]*domain.Achievement, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, achievement_key, unlocked_at
		FROM user_achievements
		WHERE user_id = $1
		ORDER BY unlocked_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("pointsRepository.ListAchievements: %w", err)
	}
	defer rows.Close()

	var achievements []*domain.Achievement
	for rows.Next() {
		var a domain.Achievement
		if err := rows.Scan(&a.ID, &a.UserID, &a.AchievementKey, &a.UnlockedAt); err != nil {
			return nil, fmt.Errorf("pointsRepository.ListAchievements scan: %w", err)
		}
		achievements = append(achievements, &a)
	}
	return achievements, rows.Err()
}

func (r *pointsRepository) UnlockAchievement(ctx context.Context, userID uuid.UUID, key string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_achievements (id, user_id, achievement_key, unlocked_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, achievement_key) DO NOTHING
	`, uuid.New(), userID, key, time.Now())
	if err != nil {
		return fmt.Errorf("pointsRepository.UnlockAchievement: %w", err)
	}
	return nil
}

func (r *pointsRepository) HasAchievement(ctx context.Context, userID uuid.UUID, key string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM user_achievements WHERE user_id = $1 AND achievement_key = $2)
	`, userID, key).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("pointsRepository.HasAchievement: %w", err)
	}
	return exists, nil
}
