package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// Save inserta un usuario nuevo (solo en registro).
func (r *userRepository) Save(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (
			id, name, email, password_hash, program_id,
			graduation_date, status, avatar_url, bio,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Name, user.Email, user.PasswordHash,
		user.ProgramID, user.GraduationDate, user.Status,
		user.AvatarURL, user.Bio, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("userRepository.Save: %w", err)
	}
	return nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

// Update actualiza los campos editables del perfil de un usuario existente.
// Solo actualiza columnas que existen en la BD actual.
func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users SET
			name       = $1,
			bio        = $2,
			avatar_url = $3,
			updated_at = $4
		WHERE id = $5
	`
	_, err := r.pool.Exec(ctx, query,
		user.Name, user.Bio, user.AvatarURL,
		time.Now(), user.ID,
	)
	if err != nil {
		return fmt.Errorf("userRepository.Update: %w", err)
	}
	return nil
}

// ─── UpdatePassword ───────────────────────────────────────────────────────────

// UpdatePassword actualiza solo el hash de contraseña.
func (r *userRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`,
		passwordHash, time.Now(), userID,
	)
	if err != nil {
		return fmt.Errorf("userRepository.UpdatePassword: %w", err)
	}
	return nil
}

// ─── FindByEmail ──────────────────────────────────────────────────────────────

// FindByEmail busca un usuario por su correo institucional.
// Solo lee las columnas base que garantizamos existen en la BD.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, program_id,
		       COALESCE(graduation_date, '0001-01-01') as graduation_date,
		       status,
		       COALESCE(avatar_url, '') as avatar_url,
		       COALESCE(bio, '') as bio,
		       created_at, updated_at
		FROM users WHERE email = $1 LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, email)
	var u domain.User
	err := row.Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.ProgramID,
		&u.GraduationDate, &u.Status, &u.AvatarURL, &u.Bio,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("usuario no encontrado con email: %s", email)
		}
		return nil, fmt.Errorf("userRepository.FindByEmail: %w", err)
	}
	return &u, nil
}

// ─── FindByID ─────────────────────────────────────────────────────────────────

// FindByID busca un usuario por su UUID.
func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, program_id,
		       COALESCE(graduation_date, '0001-01-01') as graduation_date,
		       status,
		       COALESCE(avatar_url, '') as avatar_url,
		       COALESCE(bio, '') as bio,
		       created_at, updated_at
		FROM users WHERE id = $1 LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, id)
	var u domain.User
	err := row.Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.ProgramID,
		&u.GraduationDate, &u.Status, &u.AvatarURL, &u.Bio,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("usuario no encontrado con id: %s", id)
		}
		return nil, fmt.Errorf("userRepository.FindByID: %w", err)
	}
	return &u, nil
}

// ─── FindByDocumentID ─────────────────────────────────────────────────────────

// FindByDocumentID busca un usuario por cédula de ciudadanía.
func (r *userRepository) FindByDocumentID(ctx context.Context, documentID string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, program_id,
		       graduation_date, status, avatar_url, bio,
		       document_id, phone, city, student_code, semester,
		       graduation_year, is_graduated, linkedin_url, github_url,
		       created_at, updated_at
		FROM users WHERE document_id = $1 LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, documentID)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("usuario no encontrado con cédula: %s", documentID)
		}
		return nil, fmt.Errorf("userRepository.FindByDocumentID: %w", err)
	}
	return user, nil
}

// ─── ExistsByEmail ────────────────────────────────────────────────────────────

// ExistsByEmail verifica si ya existe un usuario con ese correo.
func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("userRepository.ExistsByEmail: %w", err)
	}
	return exists, nil
}

// ─── GetRolesByUserID ─────────────────────────────────────────────────────────

// GetRolesByUserID retorna los roles asignados a un usuario.
func (r *userRepository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	query := `
		SELECT r.id, r.name
		FROM roles r
		INNER JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("userRepository.GetRolesByUserID: %w", err)
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			return nil, fmt.Errorf("userRepository.GetRolesByUserID scan: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// ─── scanUser ─────────────────────────────────────────────────────────────────

// scanUser mapea una fila de PostgreSQL a la entidad domain.User.
func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.ProgramID,
		&u.GraduationDate, &u.Status, &u.AvatarURL, &u.Bio,
		&u.DocumentID, &u.Phone, &u.City, &u.StudentCode, &u.Semester,
		&u.GraduationYear, &u.IsGraduated, &u.LinkedInURL, &u.GitHubURL,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
