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

var ErrProjectNotFound = errors.New("project no encontrado")

type projectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *projectRepository {
	return &projectRepository{pool: pool}
}
func (r *projectRepository) Create(ctx context.Context, project *domain.Project) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO projects (id, owner_id, title, description, area, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`,
		project.ID,
		project.OwnerID,
		project.Title,
		project.Description,
		project.Area,
		string(project.Status),
		project.CreatedAt,
		project.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("projectRepository.Create: %w", err)
	}
	return nil
}
func (r *projectRepository) FindByID(ctx context.Context, projectID uuid.UUID) (*domain.Project, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id, owner_id, title, description, area, status, created_at, updated_at
FROM projects
WHERE id = $1
`, projectID)
	var p domain.Project
	var status string
	if err := row.Scan(
		&p.ID,
		&p.OwnerID,
		&p.Title,
		&p.Description,
		&p.Area,
		&status,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("projectRepository.FindByID: %w", err)
	}
	p.Status = domain.ProjectStatus(status)
	return &p, nil
}
func (r *projectRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Project, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, owner_id, title, description, area, status, created_at, updated_at
FROM projects
WHERE owner_id = $1
ORDER BY created_at DESC
`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("projectRepository.ListByOwner: %w", err)
	}
	defer rows.Close()
	projects := make([]*domain.Project, 0)
	for rows.Next() {
		var p domain.Project
		var status string
		if err := rows.Scan(
			&p.ID,
			&p.OwnerID,
			&p.Title,
			&p.Description,
			&p.Area,
			&status,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("projectRepository.ListByOwner scan: %w", err)
		}
		p.Status = domain.ProjectStatus(status)
		projects = append(projects, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("projectRepository.ListByOwner rows: %w", err)
	}
	return projects, nil
}
func (r *projectRepository) UpdateStatus(ctx context.Context, projectID uuid.UUID, status domain.ProjectStatus) error {
	cmd, err := r.pool.Exec(ctx, `
UPDATE projects
SET status = $1, updated_at = NOW()
WHERE id = $2
`, string(status), projectID)
	if err != nil {
		return fmt.Errorf("projectRepository.UpdateStatus: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrProjectNotFound
	}
	return nil
}
