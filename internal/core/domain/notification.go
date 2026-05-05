package domain

import (
	"github.com/google/uuid"
	"strings"
	"time"
)

type NotificationType string

const (
	NotificationTypeMentorshipRequest  NotificationType = "mentorship_request"
	NotificationTypeMentorshipAccepted NotificationType = "mentorship_accepted"
	NotificationTypeMentorshipRejected NotificationType = "mentorship_rejected"

	NotificationTypeActividadCreada       NotificationType = "actividad_creada"
	NotificationTypeInscripcionConfirmada NotificationType = "inscripcion_confirmada"
)

type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      NotificationType
	Title     string
	Message   string
	IsRead    bool
	CreatedAt time.Time
	ReadAt    *time.Time
}

func NewNotification(userID uuid.UUID, nType NotificationType, title, message string) *Notification {
	return &Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      nType,
		Title:     strings.TrimSpace(title),
		Message:   strings.TrimSpace(message),
		IsRead:    false,
		CreatedAt: time.Now(),
	}
}
func (n *Notification) MarkAsRead() {
	if n.IsRead {
		return
	}
	now := time.Now()
	n.IsRead = true
	n.ReadAt = &now
}
