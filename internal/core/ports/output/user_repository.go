package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

// UserRepository define las operaciones de persistencia del usuario.
// La implementación concreta vive en adapters/driven/persistence/postgres.
type UserRepository interface {
	// ─── Lectura ──────────────────────────────────────────────────────────────

	// FindByID retorna un usuario completo por su UUID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// FindByIDWithRoles retorna un usuario con sus roles en una sola query (optimizado).
	FindByIDWithRoles(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// FindByEmail retorna un usuario por email (usado en auth).
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// ExistsByEmail verifica si ya existe un usuario con ese correo.
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// ExistsByDocumentID verifica si ya existe un usuario con esa cédula.
	ExistsByDocumentID(ctx context.Context, documentID string) (bool, error)

	// FindByDocumentID retorna un usuario por cédula (validación de duplicados).
	FindByDocumentID(ctx context.Context, documentID string) (*domain.User, error)

	// ─── Escritura ────────────────────────────────────────────────────────────

	// Save persiste un nuevo usuario en la base de datos.
	Save(ctx context.Context, user *domain.User) error

	// Update actualiza los campos editables de un usuario existente.
	Update(ctx context.Context, user *domain.User) error

	// UpdatePassword actualiza solo el hash de contraseña.
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error

	// GetRolesByUserID retorna los roles asignados a un usuario.
	GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)

	// ExistsProgramByID verifica si existe un programa académico por UUID.
	ExistsProgramByID(ctx context.Context, programID uuid.UUID) (bool, error)

	// FindProgramIDByName resuelve el UUID del programa a partir de su nombre.
	FindProgramIDByName(ctx context.Context, programName string) (uuid.UUID, error)

	// AssignRoleByName asigna un rol al usuario en user_roles.
	AssignRoleByName(ctx context.Context, userID uuid.UUID, roleName domain.RoleName) error
}
