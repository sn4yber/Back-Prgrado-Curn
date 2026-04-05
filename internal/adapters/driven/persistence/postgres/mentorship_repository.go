package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"time"
)

var ErrMentorshipNotFound = errors.New("mentorship no encontrado")

type mentorshipRepository struct {
	pool *pgxpool.Pool
}

func NewMentorshipRepository(pool *pgxpool.Pool) *mentorshipRepository {
	return &mentorshipRepository{pool: pool}
}
func (r *mentorshipRepository) Create(ctx context.Context, mentorship *domain.Mentorship) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO mentorships (
id, requester_id, mentor_id, project_id, status, request_message,
decision_notes, created_at, updated_at, responded_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
`,
		mentorship.ID,
		mentorship.RequesterID,
		mentorship.MentorID,
		mentorship.ProjectID,
		string(mentorship.Status),
		mentorship.RequestMessage,
		mentorship.DecisionNotes,
		mentorship.CreatedAt,
		mentorship.UpdatedAt,
		mentorship.RespondedAt,
	)
	if err != nil {
		return fmt.Errorf("mentorshipRepository.Create: %w", err)
	}
	return nil
}
func (r *mentorshipRepository) FindByID(ctx context.Context, mentorshipID uuid.UUID) (*domain.Mentorship, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id, requester_id, mentor_id, project_id, status, request_message,
       decision_notes, created_at, updated_at, responded_at
FROM mentorships
WHERE id = $1
`, mentorshipID)
	var m domain.Mentorship
	var status string
	if err := row.Scan(
		&m.ID,
		&m.RequesterID,
		&m.MentorID,
		&m.ProjectID,
		&status,
		&m.RequestMessage,
		&m.DecisionNotes,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.RespondedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("mentorshipRepository.FindByID: %w", err)
	}
	m.Status = domain.MentorshipStatus(status)
	return &m, nil
}
func (r *mentorshipRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Mentorship, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, requester_id, mentor_id, project_id, status, request_message,
       decision_notes, created_at, updated_at, responded_at
FROM mentorships
WHERE requester_id = $1 OR mentor_id = $1
ORDER BY created_at DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("mentorshipRepository.ListByUser: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.Mentorship, 0)
	for rows.Next() {
		var m domain.Mentorship
		var status string
		if err := rows.Scan(
			&m.ID,
			&m.RequesterID,
			&m.MentorID,
			&m.ProjectID,
			&status,
			&m.RequestMessage,
			&m.DecisionNotes,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.RespondedAt,
		); err != nil {
			return nil, fmt.Errorf("mentorshipRepository.ListByUser scan: %w", err)
		}
		m.Status = domain.MentorshipStatus(status)
		result = append(result, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mentorshipRepository.ListByUser rows: %w", err)
	}
	return result, nil
}
func (r *mentorshipRepository) UpdateDecision(ctx context.Context, mentorshipID uuid.UUID, status domain.MentorshipStatus, notes *string) error {
	respondedAt := time.Now()
	cmd, err := r.pool.Exec(ctx, `
UPDATE mentorships
SET status = $1, decision_notes = $2, responded_at = $3, updated_at = NOW()
WHERE id = $4
`, string(status), notes, respondedAt, mentorshipID)
	if err != nil {
		return fmt.Errorf("mentorshipRepository.UpdateDecision: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrMentorshipNotFound
	}
	return nil
}
func (r *mentorshipRepository) ExistsPendingByProjectAndMentor(ctx context.Context, projectID, mentorID uuid.UUID) (bool, error) {
	row := r.pool.QueryRow(ctx, `
SELECT EXISTS (
SELECT 1
FROM mentorships
WHERE project_id = $1 AND mentor_id = $2 AND status = 'pending'
)
`, projectID, mentorID)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("mentorshipRepository.ExistsPendingByProjectAndMentor: %w", err)
	}
	return exists, nil
}
