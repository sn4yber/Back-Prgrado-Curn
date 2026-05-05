package input

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ─── Requests ─────────────────────────────────────────────────────────────────

type CrearActividadRequest struct {
	Titulo              string   `json:"titulo"`
	Descripcion         string   `json:"descripcion"`
	Tipo                string   `json:"tipo"`
	FechaInicio         string   `json:"fecha_inicio"`
	FechaFin            *string  `json:"fecha_fin,omitempty"`
	Ubicacion           *string  `json:"ubicacion,omitempty"`
	LinkVirtual         *string  `json:"link_virtual,omitempty"`
	CapacidadMaxima     *int     `json:"capacidad_maxima,omitempty"`
	SkillsRequeridos    []string `json:"skills_requeridos,omitempty"`
	InteresesRequeridos []string `json:"intereses_requeridos,omitempty"`
}

type InscribirseRequest struct {
	Modalidad     string  `json:"modalidad"`
	Observaciones *string `json:"observaciones,omitempty"`
}

type ActualizarActividadRequest struct {
	Titulo              *string  `json:"titulo,omitempty"`
	Descripcion         *string  `json:"descripcion,omitempty"`
	FechaInicio         *string  `json:"fecha_inicio,omitempty"`
	FechaFin            *string  `json:"fecha_fin,omitempty"`
	Ubicacion           *string  `json:"ubicacion,omitempty"`
	LinkVirtual         *string  `json:"link_virtual,omitempty"`
	CapacidadMaxima     *int     `json:"capacidad_maxima,omitempty"`
	Estado              *string  `json:"estado,omitempty"`
	SkillsRequeridos    []string `json:"skills_requeridos,omitempty"`
	InteresesRequeridos []string `json:"intereses_requeridos,omitempty"`
}

// ─── Responses ────────────────────────────────────────────────────────────────

type ActividadResponse struct {
	ID                  string   `json:"id"`
	Titulo              string   `json:"titulo"`
	Descripcion         string   `json:"descripcion"`
	Tipo                string   `json:"tipo"`
	Estado              string   `json:"estado"`
	FechaInicio         string   `json:"fecha_inicio"`
	FechaFin            *string  `json:"fecha_fin,omitempty"`
	Ubicacion           *string  `json:"ubicacion,omitempty"`
	LinkVirtual         *string  `json:"link_virtual,omitempty"`
	ImagenURL           *string  `json:"imagen_url,omitempty"`
	CapacidadMaxima     *int     `json:"capacidad_maxima,omitempty"`
	CountInscritos      int      `json:"count_inscritos"`
	SkillsRequeridos    []string `json:"skills_requeridos"`
	InteresesRequeridos []string `json:"intereses_requeridos"`
	CreadorID           string   `json:"creador_id"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
}

type InscripcionActividadResponse struct {
	ID               string  `json:"id"`
	ActividadID      string  `json:"actividad_id"`
	UsuarioID        string  `json:"usuario_id"`
	Estado           string  `json:"estado"`
	Modalidad        *string `json:"modalidad,omitempty"`
	Observaciones    *string `json:"observaciones,omitempty"`
	FechaInscripcion string  `json:"fecha_inscripcion"`
}

type MisInscripcionesResponse struct {
	Inscripcion InscripcionActividadResponse `json:"inscripcion"`
	Actividad   ActividadResponse            `json:"actividad"`
}

// ─── Use Case Interface ────────────────────────────────────────────────────────

type ActividadUseCase interface {
	Crear(ctx context.Context, creadorID uuid.UUID, req CrearActividadRequest) (*ActividadResponse, error)
	ObtenerPorID(ctx context.Context, id uuid.UUID) (*ActividadResponse, error)
	Listar(ctx context.Context, tipo, estado string, limit, offset int) ([]*ActividadResponse, error)
	Actualizar(ctx context.Context, actividadID, solicitanteID uuid.UUID, req ActualizarActividadRequest) (*ActividadResponse, error)
	Eliminar(ctx context.Context, actividadID, solicitanteID uuid.UUID) error
	Inscribirse(ctx context.Context, actividadID, usuarioID uuid.UUID, req InscribirseRequest) (*InscripcionActividadResponse, error)
	CancelarInscripcion(ctx context.Context, actividadID, usuarioID uuid.UUID) error
	ObtenerMisInscripciones(ctx context.Context, usuarioID uuid.UUID) ([]*MisInscripcionesResponse, error)
	ListarInscritos(ctx context.Context, actividadID, solicitanteID uuid.UUID) ([]*InscripcionActividadResponse, error)
}

// ─── Helper de formato ────────────────────────────────────────────────────────

func FormatearFechaOpcional(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
