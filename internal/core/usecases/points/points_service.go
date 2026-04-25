package points

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

const dailyCheckinPoints = 10

type Service struct {
	pointsRepo output.PointsRepository
}

func New(pointsRepo output.PointsRepository) *Service {
	return &Service{pointsRepo: pointsRepo}
}

func (s *Service) GetMyPoints(ctx context.Context, userID uuid.UUID) (*input.PointsResponse, error) {
	p, err := s.pointsRepo.GetByUserID(ctx, userID)
	if err != nil {
		// Si no existe, devolver ceros
		return &input.PointsResponse{Points: 0, Streak: 0, LastActive: nil}, nil
	}

	var lastActive *string
	if p.LastActive != nil {
		formatted := p.LastActive.Format("2006-01-02")
		lastActive = &formatted
	}

	return &input.PointsResponse{
		Points:     p.Points,
		Streak:     p.Streak,
		LastActive: lastActive,
	}, nil
}

func (s *Service) DailyCheckin(ctx context.Context, userID uuid.UUID) (*input.CheckinResponse, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	p, err := s.pointsRepo.GetByUserID(ctx, userID)
	if err != nil {
		// Primer check-in del usuario
		p = &domain.UserPoints{
			UserID:    userID,
			Points:    0,
			Streak:    0,
			CreatedAt: now,
		}
	}

	// Verificar si ya hizo check-in hoy
	if p.LastActive != nil {
		lastDate := time.Date(p.LastActive.Year(), p.LastActive.Month(), p.LastActive.Day(), 0, 0, 0, 0, p.LastActive.Location())
		if lastDate.Equal(today) {
			return nil, apperrors.New(409, "ya registraste tu visita hoy", errors.New("duplicate checkin"))
		}

		yesterday := today.AddDate(0, 0, -1)
		if lastDate.Equal(yesterday) {
			// Día consecutivo → incrementar racha
			p.Streak++
		} else {
			// Se rompió la racha → reiniciar
			p.Streak = 1
		}
	} else {
		p.Streak = 1
	}

	p.Points += dailyCheckinPoints
	p.LastActive = &today
	p.UpdatedAt = now

	if err := s.pointsRepo.Upsert(ctx, p); err != nil {
		return nil, apperrors.New(500, "no se pudo registrar el check-in", err)
	}

	// Evaluar logros de racha y puntos
	s.evaluateAchievements(ctx, userID, p)

	message := "¡Check-in registrado!"
	if p.Streak >= 3 {
		message = "🔥 ¡Llevas " + formatStreak(p.Streak) + " días seguidos!"
	}

	return &input.CheckinResponse{
		Points:       p.Points,
		Streak:       p.Streak,
		PointsEarned: dailyCheckinPoints,
		Message:      message,
	}, nil
}

func (s *Service) GetMyAchievements(ctx context.Context, userID uuid.UUID) ([]input.AchievementResponse, error) {
	achievements, err := s.pointsRepo.ListAchievements(ctx, userID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron consultar los logros", err)
	}

	result := make([]input.AchievementResponse, 0, len(achievements))
	for _, a := range achievements {
		info, exists := domain.AchievementCatalog[a.AchievementKey]
		title := a.AchievementKey
		description := ""
		icon := "🏅"
		if exists {
			title = info.Title
			description = info.Description
			icon = info.Icon
		}
		result = append(result, input.AchievementResponse{
			Key:         a.AchievementKey,
			Title:       title,
			Description: description,
			Icon:        icon,
			UnlockedAt:  a.UnlockedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}

// evaluateAchievements verifica y desbloquea logros si aplica.
func (s *Service) evaluateAchievements(ctx context.Context, userID uuid.UUID, p *domain.UserPoints) {
	checks := []struct {
		key       string
		condition bool
	}{
		{domain.AchievementStreak3, p.Streak >= 3},
		{domain.AchievementStreak7, p.Streak >= 7},
		{domain.AchievementStreak30, p.Streak >= 30},
		{domain.AchievementPoints100, p.Points >= 100},
		{domain.AchievementPoints500, p.Points >= 500},
	}

	for _, c := range checks {
		if c.condition {
			has, _ := s.pointsRepo.HasAchievement(ctx, userID, c.key)
			if !has {
				_ = s.pointsRepo.UnlockAchievement(ctx, userID, c.key)
			}
		}
	}
}

func formatStreak(n int) string {
	if n < 10 {
		return string(rune('0'+n)) + ""
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
