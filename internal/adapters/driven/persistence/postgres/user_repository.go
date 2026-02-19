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

// userRepository implementa output.UserRepository usando PostgreSQL.
type userRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository construye el repositorio con el pool inyectado.
func NewUserRepository(pool *pgxpool.Pool) *userRepository {
	return &userRepository{pool: pool}
}

// ─── Save ─────────────────────────────────────────────────────────────────────

// Save inserta un usuario nuevo. No hace upsert — usa ExistsByEmail antes de llamar esto.
func (r *userRepository) Save(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, name, email, password_hash, program_id, graduation_date, status, avatar_url, bio, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.ProgramID,
		user.GraduationDate,
		user.Status,
		user.AvatarURL,
		user.Bio,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("userRepository.Save: %w", err)
	}

	return nil
}

// ─── FindByEmail ──────────────────────────────────────────────────────────────

// FindByEmail busca un usuario por su correo institucional.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, program_id, graduation_date,
		       status, avatar_url, bio, created_at, updated_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	row := r.pool.QueryRow(ctx, query, email)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("usuario no encontrado con email: %s", email)
		}
		return nil, fmt.Errorf("userRepository.FindByEmail: %w", err)
	}

	return user, nil
}

// ─── FindByID ─────────────────────────────────────────────────────────────────

// FindByID busca un usuario por su UUID.
func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, program_id, graduation_date,
		       status, avatar_url, bio, created_at, updated_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	row := r.pool.QueryRow(ctx, query, id)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("usuario no encontrado con id: %s", id)
		}
		return nil, fmt.Errorf("userRepository.FindByID: %w", err)
	}

	return user, nil
}

// ─── ExistsByEmail ────────────────────────────────────────────────────────────

// ExistsByEmail verifica si ya existe un usuario con ese correo.
// Más eficiente que FindByEmail porque no trae todos los campos.
func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("userRepository.ExistsByEmail: %w", err)
	}

	return exists, nil
}

// ─── scanUser ────────────────────────────────────────────────────────────────

// scanUser mapea una fila de PostgreSQL a la entidad domain.User.
// Centralizado aquí para no repetir el mapeo en cada función.
func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.PasswordHash,
		&u.ProgramID,
		&u.GraduationDate,
		&u.Status,
		&u.AvatarURL,
		&u.Bio,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
