package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID         uuid.UUID
	PostID     uuid.UUID
	AuthorID   uuid.UUID
	AuthorName *string
	Content    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (c *Comment) Normalize() {
	c.Content = strings.TrimSpace(c.Content)
}
