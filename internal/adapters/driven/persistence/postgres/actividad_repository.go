package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
)

var ErrActividadNotFound = errors.New("actividad no encontrada")

// ─── actividadRepository ──────────────────────────────────────────────────────

type actividadRepository struct {
	pool *pgxpool.Pool
}

func NewActividadRepository(pool *pgxpool.Pool) *actividadRepository {
	return &actividadRepository{pool: pool}
}

func (r *actividadRepository) Crear(ctx context.Context, a *domain.Actividad) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO actividades (
			id, titulo, descripcion, tipo, estado,
			fecha_inicio, fecha_fin, ubicacion, link_virtual, imagen_url,
			capacidad_maxima, count_inscritos,
			skills_requeridos, intereses_requeridos,
			creador_id, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,$10,
			$11,$12,
			$13,$14,
			$15,$16,$17
		)`,
		a.ID, a.Titulo, a.Descripcion, string(a.Tipo), string(a.Estado),
		a.FechaInicio, a.FechaFin, a.Ubicacion, a.LinkVirtual, a.ImagenURL,
		a.CapacidadMaxima, a.CountInscritos,
		a.SkillsRequeridos, a.InteresesRequeridos,
		a.CreadorID, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("actividadRepository.Crear: %w", err)
	}
	return nil
}

func (r *actividadRepository) ObtenerPorID(ctx context.Context, id uuid.UUID) (*domain.Actividad, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, titulo, descripcion, tipo, estado,
		       fecha_inicio, fecha_fin, ubicacion, link_virtual, imagen_url,
		       capacidad_maxima, count_inscritos,
		       skills_requeridos, intereses_requeridos,
		       creador_id, created_at, updated_at
		FROM actividades
		WHERE id = $1
	`, id)
	return scanActividad(row)
}

func (r *actividadRepository) Listar(ctx context.Context, filtros output.FiltrosActividad) ([]*domain.Actividad, error) {
	query := `
		SELECT id, titulo, descripcion, tipo, estado,
		       fecha_inicio, fecha_fin, ubicacion, link_virtual, imagen_url,
		       capacidad_maxima, count_inscritos,
		       skills_requeridos, intereses_requeridos,
		       creador_id, created_at, updated_at
		FROM actividades
		WHERE 1=1`
	args := []any{}
	n := 1

	if filtros.Tipo != "" {
		query += fmt.Sprintf(" AND tipo = $%d", n)
		args = append(args, filtros.Tipo)
		n++
	}
	if filtros.Estado != "" {
		query += fmt.Sprintf(" AND estado = $%d", n)
		args = append(args, filtros.Estado)
		n++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", n, n+1)
	args = append(args, filtros.Limit, filtros.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("actividadRepository.Listar: %w", err)
	}
	defer rows.Close()

	var actividades []*domain.Actividad
	for rows.Next() {
		a, err := scanActividadRows(rows)
		if err != nil {
			return nil, fmt.Errorf("actividadRepository.Listar scan: %w", err)
		}
		actividades = append(actividades, a)
	}
	return actividades, rows.Err()
}

func (r *actividadRepository) Actualizar(ctx context.Context, a *domain.Actividad) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE actividades SET
			titulo              = $1,
			descripcion         = $2,
			tipo                = $3,
			estado              = $4,
			fecha_inicio        = $5,
			fecha_fin           = $6,
			ubicacion           = $7,
			link_virtual        = $8,
			capacidad_maxima    = $9,
			skills_requeridos   = $10,
			intereses_requeridos= $11,
			updated_at          = $12
		WHERE id = $13`,
		a.Titulo, a.Descripcion, string(a.Tipo), string(a.Estado),
		a.FechaInicio, a.FechaFin, a.Ubicacion, a.LinkVirtual,
		a.CapacidadMaxima, a.SkillsRequeridos, a.InteresesRequeridos,
		a.UpdatedAt, a.ID,
	)
	if err != nil {
		return fmt.Errorf("actividadRepository.Actualizar: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrActividadNotFound
	}
	return nil
}

func (r *actividadRepository) Eliminar(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM actividades WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("actividadRepository.Eliminar: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrActividadNotFound
	}
	return nil
}

func (r *actividadRepository) IncrementarInscritos(ctx context.Context, actividadID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE actividades SET count_inscritos = count_inscritos + 1 WHERE id = $1`,
		actividadID,
	)
	return err
}

func (r *actividadRepository) DecrementarInscritos(ctx context.Context, actividadID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE actividades SET count_inscritos = GREATEST(count_inscritos - 1, 0) WHERE id = $1`,
		actividadID,
	)
	return err
}

// ─── inscripcionActividadRepository ──────────────────────────────────────────

type inscripcionActividadRepository struct {
	pool *pgxpool.Pool
}

func NewInscripcionActividadRepository(pool *pgxpool.Pool) *inscripcionActividadRepository {
	return &inscripcionActividadRepository{pool: pool}
}

func (r *inscripcionActividadRepository) Inscribir(ctx context.Context, ins *domain.InscripcionActividad) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO inscripciones_actividades (id, actividad_id, usuario_id, estado, modalidad, observaciones, fecha_inscripcion)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (actividad_id, usuario_id)
		DO UPDATE SET estado = $4, modalidad = $5, observaciones = $6, fecha_inscripcion = $7
	`, ins.ID, ins.ActividadID, ins.UsuarioID, string(ins.Estado), ins.Modalidad, ins.Observaciones, ins.FechaInscripcion)
	if err != nil {
		return fmt.Errorf("inscripcionActividadRepository.Inscribir: %w", err)
	}
	return nil
}

func (r *inscripcionActividadRepository) CancelarInscripcion(ctx context.Context, actividadID, usuarioID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE inscripciones_actividades
		SET estado = 'cancelado'
		WHERE actividad_id = $1 AND usuario_id = $2
	`, actividadID, usuarioID)
	if err != nil {
		return fmt.Errorf("inscripcionActividadRepository.CancelarInscripcion: %w", err)
	}
	return nil
}

func (r *inscripcionActividadRepository) ObtenerInscripcionUsuario(ctx context.Context, actividadID, usuarioID uuid.UUID) (*domain.InscripcionActividad, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, actividad_id, usuario_id, estado, modalidad, observaciones, fecha_inscripcion
		FROM inscripciones_actividades
		WHERE actividad_id = $1 AND usuario_id = $2
	`, actividadID, usuarioID)
	var ins domain.InscripcionActividad
	var estado string
	err := row.Scan(&ins.ID, &ins.ActividadID, &ins.UsuarioID, &estado, &ins.Modalidad, &ins.Observaciones, &ins.FechaInscripcion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("inscripcionActividadRepository.ObtenerInscripcionUsuario: %w", err)
	}
	ins.Estado = domain.EstadoInscripcion(estado)
	return &ins, nil
}

func (r *inscripcionActividadRepository) ListarPorActividad(ctx context.Context, actividadID uuid.UUID) ([]*domain.InscripcionActividad, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, actividad_id, usuario_id, estado, modalidad, observaciones, fecha_inscripcion
		FROM inscripciones_actividades
		WHERE actividad_id = $1
		ORDER BY fecha_inscripcion ASC
	`, actividadID)
	if err != nil {
		return nil, fmt.Errorf("inscripcionActividadRepository.ListarPorActividad: %w", err)
	}
	defer rows.Close()
	return scanInscripciones(rows)
}

func (r *inscripcionActividadRepository) ListarPorUsuario(ctx context.Context, usuarioID uuid.UUID) ([]*domain.InscripcionActividad, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, actividad_id, usuario_id, estado, modalidad, observaciones, fecha_inscripcion
		FROM inscripciones_actividades
		WHERE usuario_id = $1
		ORDER BY fecha_inscripcion DESC
	`, usuarioID)
	if err != nil {
		return nil, fmt.Errorf("inscripcionActividadRepository.ListarPorUsuario: %w", err)
	}
	defer rows.Close()
	return scanInscripciones(rows)
}

// ─── Helpers de scan ─────────────────────────────────────────────────────────

func scanActividad(row pgx.Row) (*domain.Actividad, error) {
	var a domain.Actividad
	var tipo, estado string
	err := row.Scan(
		&a.ID, &a.Titulo, &a.Descripcion, &tipo, &estado,
		&a.FechaInicio, &a.FechaFin, &a.Ubicacion, &a.LinkVirtual, &a.ImagenURL,
		&a.CapacidadMaxima, &a.CountInscritos,
		&a.SkillsRequeridos, &a.InteresesRequeridos,
		&a.CreadorID, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scanActividad: %w", err)
	}
	a.Tipo = domain.TipoActividad(tipo)
	a.Estado = domain.EstadoActividad(estado)
	return &a, nil
}

func scanActividadRows(rows pgx.Rows) (*domain.Actividad, error) {
	var a domain.Actividad
	var tipo, estado string
	err := rows.Scan(
		&a.ID, &a.Titulo, &a.Descripcion, &tipo, &estado,
		&a.FechaInicio, &a.FechaFin, &a.Ubicacion, &a.LinkVirtual, &a.ImagenURL,
		&a.CapacidadMaxima, &a.CountInscritos,
		&a.SkillsRequeridos, &a.InteresesRequeridos,
		&a.CreadorID, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	a.Tipo = domain.TipoActividad(tipo)
	a.Estado = domain.EstadoActividad(estado)
	return &a, nil
}

func scanInscripciones(rows pgx.Rows) ([]*domain.InscripcionActividad, error) {
	var resultado []*domain.InscripcionActividad
	for rows.Next() {
		var ins domain.InscripcionActividad
		var estado string
		if err := rows.Scan(&ins.ID, &ins.ActividadID, &ins.UsuarioID, &estado, &ins.Modalidad, &ins.Observaciones, &ins.FechaInscripcion); err != nil {
			return nil, err
		}
		ins.Estado = domain.EstadoInscripcion(estado)
		resultado = append(resultado, &ins)
	}
	return resultado, rows.Err()
}
