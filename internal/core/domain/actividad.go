package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── ENUMs ───────────────────────────────────────────────────────────────────

type TipoActividad string

const (
	TipoActividadCharla       TipoActividad = "charla"
	TipoActividadAuditoria    TipoActividad = "auditoria"
	TipoActividadEvento       TipoActividad = "evento"
	TipoActividadConvocatoria TipoActividad = "convocatoria"
	TipoActividadDeportivo    TipoActividad = "deportivo"
	TipoActividadOtro         TipoActividad = "otro"
)

type EstadoActividad string

const (
	EstadoActividadPublicada  EstadoActividad = "publicada"
	EstadoActividadCancelada  EstadoActividad = "cancelada"
	EstadoActividadCompletada EstadoActividad = "completada"
)

type EstadoInscripcion string

const (
	EstadoInscripcionInscrito  EstadoInscripcion = "inscrito"
	EstadoInscripcionCancelado EstadoInscripcion = "cancelado"
	EstadoInscripcionAsistido  EstadoInscripcion = "asistido"
)

// ─── Errores de dominio ───────────────────────────────────────────────────────

var (
	ErrActividadSinCupo         = errors.New("la actividad no tiene cupo disponible")
	ErrActividadNoPublicada     = errors.New("solo se puede inscribir en actividades publicadas")
	ErrActividadYaInscrito      = errors.New("ya estás inscrito en esta actividad")
	ErrActividadTransicionInval = errors.New("transición de estado inválida para la actividad")
)

// ─── Entidades ────────────────────────────────────────────────────────────────

type Actividad struct {
	ID          uuid.UUID
	Titulo      string
	Descripcion string
	Tipo        TipoActividad
	Estado      EstadoActividad

	FechaInicio  time.Time
	FechaFin     *time.Time
	Ubicacion    *string
	LinkVirtual  *string
	ImagenURL    *string

	CapacidadMaxima   *int
	CountInscritos    int

	SkillsRequeridos    []string
	InteresesRequeridos []string

	CreadorID uuid.UUID

	CreatedAt time.Time
	UpdatedAt time.Time
}

type InscripcionActividad struct {
	ID          uuid.UUID
	ActividadID uuid.UUID
	UsuarioID   uuid.UUID
	Estado      EstadoInscripcion
	FechaInscripcion time.Time
}

// ─── Constructores ────────────────────────────────────────────────────────────

func NuevaActividad(
	creadorID uuid.UUID,
	titulo, descripcion string,
	tipo TipoActividad,
	fechaInicio time.Time,
) *Actividad {
	ahora := time.Now()
	return &Actividad{
		ID:                  uuid.New(),
		Titulo:              strings.TrimSpace(titulo),
		Descripcion:         strings.TrimSpace(descripcion),
		Tipo:                tipo,
		Estado:              EstadoActividadPublicada,
		FechaInicio:         fechaInicio,
		SkillsRequeridos:    []string{},
		InteresesRequeridos: []string{},
		CreadorID:           creadorID,
		CreatedAt:           ahora,
		UpdatedAt:           ahora,
	}
}

func NuevaInscripcion(actividadID, usuarioID uuid.UUID) *InscripcionActividad {
	return &InscripcionActividad{
		ID:               uuid.New(),
		ActividadID:      actividadID,
		UsuarioID:        usuarioID,
		Estado:           EstadoInscripcionInscrito,
		FechaInscripcion: time.Now(),
	}
}

// ─── Reglas de negocio ────────────────────────────────────────────────────────

func (a *Actividad) TieneCupo() bool {
	if a.CapacidadMaxima == nil {
		return true
	}
	return a.CountInscritos < *a.CapacidadMaxima
}

func (a *Actividad) EsPublicada() bool {
	return a.Estado == EstadoActividadPublicada
}

func (a *Actividad) PuedeInscribirse() error {
	if !a.EsPublicada() {
		return ErrActividadNoPublicada
	}
	if !a.TieneCupo() {
		return ErrActividadSinCupo
	}
	return nil
}
