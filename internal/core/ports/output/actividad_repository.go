package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type FiltrosActividad struct {
	Tipo   string
	Estado string
	Limit  int
	Offset int
}

type ActividadRepository interface {
	Crear(ctx context.Context, actividad *domain.Actividad) error
	ObtenerPorID(ctx context.Context, id uuid.UUID) (*domain.Actividad, error)
	Listar(ctx context.Context, filtros FiltrosActividad) ([]*domain.Actividad, error)
	Actualizar(ctx context.Context, actividad *domain.Actividad) error
	Eliminar(ctx context.Context, id uuid.UUID) error
	IncrementarInscritos(ctx context.Context, actividadID uuid.UUID) error
	DecrementarInscritos(ctx context.Context, actividadID uuid.UUID) error
}

type InscripcionActividadRepository interface {
	Inscribir(ctx context.Context, inscripcion *domain.InscripcionActividad) error
	CancelarInscripcion(ctx context.Context, actividadID, usuarioID uuid.UUID) error
	ObtenerInscripcionUsuario(ctx context.Context, actividadID, usuarioID uuid.UUID) (*domain.InscripcionActividad, error)
	ListarPorActividad(ctx context.Context, actividadID uuid.UUID) ([]*domain.InscripcionActividad, error)
	ListarPorUsuario(ctx context.Context, usuarioID uuid.UUID) ([]*domain.InscripcionActividad, error)
}
