package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserPoints representa los puntos y racha de un usuario.
type UserPoints struct {
	UserID     uuid.UUID
	Points     int
	Streak     int
	LastActive *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Achievement representa un logro desbloqueado por un usuario.
type Achievement struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	AchievementKey string
	UnlockedAt     time.Time
}

// Claves de logros predefinidos.
const (
	AchievementFirstPost       = "first_post"
	AchievementFirstConnection = "first_connection"
	AchievementStreak3         = "streak_3"
	AchievementStreak7         = "streak_7"
	AchievementStreak30        = "streak_30"
	AchievementPoints100       = "points_100"
	AchievementPoints500       = "points_500"
)

// AchievementInfo contiene metadatos de un logro para el frontend.
var AchievementCatalog = map[string]struct {
	Title       string
	Description string
	Icon        string
}{
	AchievementFirstPost:       {"Primera Publicación", "Creaste tu primera publicación en Conecta Curn", "📝"},
	AchievementFirstConnection: {"Primera Conexión", "Te conectaste con otro miembro de la comunidad", "🤝"},
	AchievementStreak3:         {"Racha de 3 días", "Visitaste Conecta Curn 3 días seguidos", "🔥"},
	AchievementStreak7:         {"Racha de 7 días", "Visitaste Conecta Curn 7 días seguidos", "⚡"},
	AchievementStreak30:        {"Racha de 30 días", "Visitaste Conecta Curn 30 días seguidos", "🏆"},
	AchievementPoints100:       {"100 Puntos", "Acumulaste 100 puntos de actividad", "💯"},
	AchievementPoints500:       {"500 Puntos", "Acumulaste 500 puntos de actividad", "🌟"},
}
