package domain

import (
	"time"

	"github.com/google/uuid"
)

// ─── ENUMs del dominio ────────────────────────────────────────────────────────

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBanned   UserStatus = "banned"
)

type RoleName string

const (
	RoleEstudiante     RoleName = "estudiante"
	RoleEgresado       RoleName = "egresado"
	RoleAdministrativo RoleName = "administrativo"
	RoleAdmin          RoleName = "admin"
)

// ─── Entidades ────────────────────────────────────────────────────────────────

// Role representa un rol del sistema.
type Role struct {
	ID   int
	Name RoleName
}

// User es la entidad central del dominio.
// No contiene lógica de base de datos ni de HTTP — solo el negocio.
type User struct {
	ID             uuid.UUID
	Name           string
	Email          string
	PasswordHash   string
	ProgramID      uuid.UUID
	GraduationDate *time.Time // nil = estudiante activo
	Status         UserStatus
	AvatarURL      *string
	Bio            *string
	Roles          []Role
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ─── Reglas de negocio ────────────────────────────────────────────────────────

// IsActive verifica que el usuario pueda operar en la plataforma.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsEgresado verifica si el usuario tiene rol de egresado.
func (u *User) IsEgresado() bool {
	for _, r := range u.Roles {
		if r.Name == RoleEgresado {
			return true
		}
	}
	return false
}

// HasRole verifica si el usuario tiene un rol específico.
func (u *User) HasRole(role RoleName) bool {
	for _, r := range u.Roles {
		if r.Name == role {
			return true
		}
	}
	return false
}
