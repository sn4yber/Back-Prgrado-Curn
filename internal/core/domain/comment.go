package domain

import (
	"github.com/google/uuid"
	"strings"
	"time"
)

type Comment struct {
	ID        uuid.UUID
	PostID    uuid.UUID
	AuthorID  uuid.UUID
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *Comment) Normalize() {
	c.Content = strings.TrimSpace(c.Content)
}
