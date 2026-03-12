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
			document_id, phone, city,
			student_code, semester, graduation_year, is_graduated,
			linkedin_url, github_url,
			created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,
			$10,$11,$12,
			$13,$14,$15,$16,
			$17,$18,
			$19,$20
		)
	`
	_, err := r.pool.Exec(ctx, query,
		user.ID, user.Name, user.Email, user.PasswordHash,
		user.ProgramID, user.GraduationDate, user.Status,
		user.AvatarURL, user.Bio,
		user.DocumentID, user.Phone, user.City,
		user.StudentCode, user.Semester, user.GraduationYear, user.IsGraduated,
		user.LinkedInURL, user.GitHubURL,
		user.CreatedAt, user.UpdatedAt,
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
			name            = $1,
			bio             = $2,
			avatar_url      = $3,
			document_id     = $4,
			phone           = $5,
			city            = $6,
			student_code    = $7,
			semester        = $8,
			graduation_year = $9,
			is_graduated    = $10,
			linkedin_url    = $11,
			github_url      = $12,
			updated_at      = $13
		WHERE id = $14
	`
	cmd, err := r.pool.Exec(ctx, query,
		user.Name, user.Bio, user.AvatarURL,
		user.DocumentID, user.Phone, user.City,
		user.StudentCode, user.Semester, user.GraduationYear, user.IsGraduated,
		user.LinkedInURL, user.GitHubURL,
		time.Now(), user.ID,
	)
	if err != nil {
		return fmt.Errorf("userRepository.Update: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("userRepository.Update: usuario no encontrado")
	}
	return nil
}

// ─── UpdatePassword ───────────────────────────────────────────────────────────

// UpdatePassword actualiza solo el hash de contraseña.
func (r *userRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	cmd, err := r.pool.Exec(ctx,
		`UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3`,
		passwordHash, time.Now(), userID,
	)
	if err != nil {
		return fmt.Errorf("userRepository.UpdatePassword: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("userRepository.UpdatePassword: usuario no encontrado")
	}
	return nil
}

// ─── FindByEmail ──────────────────────────────────────────────────────────────

// FindByEmail busca un usuario por su correo institucional.
// Solo lee las columnas base que garantizamos existen en la BD.
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, program_id,
		       graduation_date,
		       status,
		       avatar_url,
		       bio,
		       document_id, phone, city, student_code, semester,
		       graduation_year, is_graduated, linkedin_url, github_url,
		       created_at, updated_at
		FROM users WHERE email = $1 LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, email)
	var u domain.User
	err := row.Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.ProgramID,
		&u.GraduationDate, &u.Status, &u.AvatarURL, &u.Bio,
		&u.DocumentID, &u.Phone, &u.City, &u.StudentCode, &u.Semester,
		&u.GraduationYear, &u.IsGraduated, &u.LinkedInURL, &u.GitHubURL,
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
		       graduation_date,
		       status,
		       avatar_url,
		       bio,
		       document_id, phone, city, student_code, semester,
		       graduation_year, is_graduated, linkedin_url, github_url,
		       created_at, updated_at
		FROM users WHERE id = $1 LIMIT 1
	`
	row := r.pool.QueryRow(ctx, query, id)
	var u domain.User
	err := row.Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.ProgramID,
		&u.GraduationDate, &u.Status, &u.AvatarURL, &u.Bio,
		&u.DocumentID, &u.Phone, &u.City, &u.StudentCode, &u.Semester,
		&u.GraduationYear, &u.IsGraduated, &u.LinkedInURL, &u.GitHubURL,
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

// ─── FindByIDWithRoles ────────────────────────────────────────────────────────

// FindByIDWithRoles busca un usuario con sus roles en una sola query (optimizado).
// Elimina el problema N+1 al hacer JOIN en vez de 2 queries separadas.
func (r *userRepository) FindByIDWithRoles(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT
			u.id, u.name, u.email, u.password_hash, u.program_id,
			u.graduation_date,
			u.status,
			u.avatar_url,
			u.bio,
			u.document_id, u.phone, u.city, u.student_code, u.semester,
			u.graduation_year, u.is_graduated, u.linkedin_url, u.github_url,
			u.created_at, u.updated_at,
			COALESCE(array_agg(r.id) FILTER (WHERE r.id IS NOT NULL), '{}') as role_ids,
			COALESCE(array_agg(r.name) FILTER (WHERE r.id IS NOT NULL), '{}') as role_names
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE u.id = $1
		GROUP BY u.id
		LIMIT 1
	`

	var u domain.User
	var roleIDs []int
	var roleNames []string

	row := r.pool.QueryRow(ctx, query, id)
	err := row.Scan(
		&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.ProgramID,
		&u.GraduationDate,
		&u.Status,
		&u.AvatarURL,
		&u.Bio,
		&u.DocumentID, &u.Phone, &u.City, &u.StudentCode, &u.Semester,
		&u.GraduationYear, &u.IsGraduated, &u.LinkedInURL, &u.GitHubURL,
		&u.CreatedAt, &u.UpdatedAt,
		&roleIDs, &roleNames,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("usuario no encontrado con id: %s", id)
		}
		return nil, fmt.Errorf("userRepository.FindByIDWithRoles: %w", err)
	}

	u.Roles = make([]domain.Role, 0, len(roleIDs))
	for i := range roleIDs {
		u.Roles = append(u.Roles, domain.Role{
			ID:   roleIDs[i],
			Name: domain.RoleName(roleNames[i]),
		})
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
