package actividad

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type Servicio struct {
	actividadRepo   output.ActividadRepository
	inscripcionRepo output.InscripcionActividadRepository
	userRepo        output.UserRepository
	notificacionRepo output.NotificationRepository
}

func Nuevo(
	actividadRepo output.ActividadRepository,
	inscripcionRepo output.InscripcionActividadRepository,
	userRepo output.UserRepository,
	notificacionRepo output.NotificationRepository,
) *Servicio {
	return &Servicio{
		actividadRepo:    actividadRepo,
		inscripcionRepo:  inscripcionRepo,
		userRepo:         userRepo,
		notificacionRepo: notificacionRepo,
	}
}

// ─── Crear ────────────────────────────────────────────────────────────────────

func (s *Servicio) Crear(ctx context.Context, creadorID uuid.UUID, req input.CrearActividadRequest) (*input.ActividadResponse, error) {
	titulo := strings.TrimSpace(req.Titulo)
	if titulo == "" {
		return nil, apperrors.New(400, "el título es requerido", nil)
	}
	descripcion := strings.TrimSpace(req.Descripcion)
	if descripcion == "" {
		return nil, apperrors.New(400, "la descripción es requerida", nil)
	}
	tipo := domain.TipoActividad(req.Tipo)
	if !tipoValido(tipo) {
		return nil, apperrors.New(400, "tipo de actividad inválido", nil)
	}
	fechaInicio, err := time.Parse(time.RFC3339, req.FechaInicio)
	if err != nil {
		return nil, apperrors.New(400, "fecha_inicio inválida, usa formato RFC3339", err)
	}

	act := domain.NuevaActividad(creadorID, titulo, descripcion, tipo, fechaInicio)

	if req.FechaFin != nil {
		fechaFin, err := time.Parse(time.RFC3339, *req.FechaFin)
		if err != nil {
			return nil, apperrors.New(400, "fecha_fin inválida, usa formato RFC3339", err)
		}
		act.FechaFin = &fechaFin
	}
	act.Ubicacion = req.Ubicacion
	act.LinkVirtual = req.LinkVirtual
	act.CapacidadMaxima = req.CapacidadMaxima
	if len(req.SkillsRequeridos) > 0 {
		act.SkillsRequeridos = req.SkillsRequeridos
	}
	if len(req.InteresesRequeridos) > 0 {
		act.InteresesRequeridos = req.InteresesRequeridos
	}

	if err := s.actividadRepo.Crear(ctx, act); err != nil {
		return nil, apperrors.New(500, "no se pudo crear la actividad", err)
	}

	go s.enviarNotificacionesAsync(act)

	return toActividadResponse(act), nil
}

// ─── ObtenerPorID ─────────────────────────────────────────────────────────────

func (s *Servicio) ObtenerPorID(ctx context.Context, id uuid.UUID) (*input.ActividadResponse, error) {
	act, err := s.actividadRepo.ObtenerPorID(ctx, id)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo obtener la actividad", err)
	}
	if act == nil {
		return nil, apperrors.New(404, "actividad no encontrada", nil)
	}
	return toActividadResponse(act), nil
}

// ─── Listar ───────────────────────────────────────────────────────────────────

func (s *Servicio) Listar(ctx context.Context, tipo, estado string, limit, offset int) ([]*input.ActividadResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	filtros := output.FiltrosActividad{
		Tipo:   tipo,
		Estado: estado,
		Limit:  limit,
		Offset: offset,
	}
	actividades, err := s.actividadRepo.Listar(ctx, filtros)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo listar actividades", err)
	}
	resp := make([]*input.ActividadResponse, 0, len(actividades))
	for _, a := range actividades {
		resp = append(resp, toActividadResponse(a))
	}
	return resp, nil
}

// ─── Actualizar ───────────────────────────────────────────────────────────────

func (s *Servicio) Actualizar(ctx context.Context, actividadID, solicitanteID uuid.UUID, req input.ActualizarActividadRequest) (*input.ActividadResponse, error) {
	act, err := s.actividadRepo.ObtenerPorID(ctx, actividadID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo obtener la actividad", err)
	}
	if act == nil {
		return nil, apperrors.New(404, "actividad no encontrada", nil)
	}

	usuario, err := s.userRepo.FindByIDWithRoles(ctx, solicitanteID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	esAdmin := usuario.HasRole(domain.RoleAdmin) || usuario.HasRole(domain.RoleAdministrativo)
	if act.CreadorID != solicitanteID && !esAdmin {
		return nil, apperrors.New(403, "solo el creador o un administrador puede editar esta actividad", nil)
	}

	if req.Titulo != nil {
		act.Titulo = strings.TrimSpace(*req.Titulo)
	}
	if req.Descripcion != nil {
		act.Descripcion = strings.TrimSpace(*req.Descripcion)
	}
	if req.FechaInicio != nil {
		fi, err := time.Parse(time.RFC3339, *req.FechaInicio)
		if err != nil {
			return nil, apperrors.New(400, "fecha_inicio inválida", err)
		}
		act.FechaInicio = fi
	}
	if req.FechaFin != nil {
		ff, err := time.Parse(time.RFC3339, *req.FechaFin)
		if err != nil {
			return nil, apperrors.New(400, "fecha_fin inválida", err)
		}
		act.FechaFin = &ff
	}
	if req.Ubicacion != nil {
		act.Ubicacion = req.Ubicacion
	}
	if req.LinkVirtual != nil {
		act.LinkVirtual = req.LinkVirtual
	}
	if req.CapacidadMaxima != nil {
		act.CapacidadMaxima = req.CapacidadMaxima
	}
	if req.Estado != nil {
		est := domain.EstadoActividad(*req.Estado)
		if !estadoValido(est) {
			return nil, apperrors.New(400, "estado de actividad inválido", nil)
		}
		act.Estado = est
	}
	if req.SkillsRequeridos != nil {
		act.SkillsRequeridos = req.SkillsRequeridos
	}
	if req.InteresesRequeridos != nil {
		act.InteresesRequeridos = req.InteresesRequeridos
	}
	act.UpdatedAt = time.Now()

	if err := s.actividadRepo.Actualizar(ctx, act); err != nil {
		return nil, apperrors.New(500, "no se pudo actualizar la actividad", err)
	}
	return toActividadResponse(act), nil
}

// ─── Eliminar ─────────────────────────────────────────────────────────────────

func (s *Servicio) Eliminar(ctx context.Context, actividadID, solicitanteID uuid.UUID) error {
	act, err := s.actividadRepo.ObtenerPorID(ctx, actividadID)
	if err != nil {
		return apperrors.New(500, "no se pudo obtener la actividad", err)
	}
	if act == nil {
		return apperrors.New(404, "actividad no encontrada", nil)
	}

	usuario, err := s.userRepo.FindByIDWithRoles(ctx, solicitanteID)
	if err != nil {
		return apperrors.ErrUserNotFound
	}
	esAdmin := usuario.HasRole(domain.RoleAdmin) || usuario.HasRole(domain.RoleAdministrativo)
	if act.CreadorID != solicitanteID && !esAdmin {
		return apperrors.New(403, "solo el creador o un administrador puede eliminar esta actividad", nil)
	}

	if err := s.actividadRepo.Eliminar(ctx, actividadID); err != nil {
		return apperrors.New(500, "no se pudo eliminar la actividad", err)
	}
	return nil
}

// ─── Inscribirse ──────────────────────────────────────────────────────────────

func (s *Servicio) Inscribirse(ctx context.Context, actividadID, usuarioID uuid.UUID, req input.InscribirseRequest) (*input.InscripcionActividadResponse, error) {
	act, err := s.actividadRepo.ObtenerPorID(ctx, actividadID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo obtener la actividad", err)
	}
	if act == nil {
		return nil, apperrors.New(404, "actividad no encontrada", nil)
	}

	if err := act.PuedeInscribirse(); err != nil {
		switch err {
		case domain.ErrActividadNoPublicada:
			return nil, apperrors.New(422, err.Error(), nil)
		case domain.ErrActividadSinCupo:
			return nil, apperrors.New(409, err.Error(), nil)
		}
		return nil, apperrors.New(422, err.Error(), nil)
	}

	existente, err := s.inscripcionRepo.ObtenerInscripcionUsuario(ctx, actividadID, usuarioID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo verificar inscripción existente", err)
	}
	if existente != nil && existente.Estado == domain.EstadoInscripcionInscrito {
		return nil, apperrors.New(409, domain.ErrActividadYaInscrito.Error(), nil)
	}

	var modalidad *string
	if req.Modalidad != "" {
		modalidad = &req.Modalidad
	}
	inscripcion := domain.NuevaInscripcion(actividadID, usuarioID, modalidad, req.Observaciones)
	if err := s.inscripcionRepo.Inscribir(ctx, inscripcion); err != nil {
		return nil, apperrors.New(500, "no se pudo registrar la inscripción", err)
	}

	_ = s.actividadRepo.IncrementarInscritos(ctx, actividadID)

	if s.notificacionRepo != nil {
		n := domain.NewNotification(
			usuarioID,
			domain.NotificationTypeInscripcionConfirmada,
			"Inscripción confirmada",
			"Tu inscripción en '"+act.Titulo+"' fue confirmada.",
		)
		_ = s.notificacionRepo.Create(ctx, n)
	}

	return toInscripcionResponse(inscripcion), nil
}

// ─── CancelarInscripcion ──────────────────────────────────────────────────────

func (s *Servicio) CancelarInscripcion(ctx context.Context, actividadID, usuarioID uuid.UUID) error {
	existente, err := s.inscripcionRepo.ObtenerInscripcionUsuario(ctx, actividadID, usuarioID)
	if err != nil {
		return apperrors.New(500, "no se pudo verificar la inscripción", err)
	}
	if existente == nil || existente.Estado != domain.EstadoInscripcionInscrito {
		return apperrors.New(404, "no tienes una inscripción activa en esta actividad", nil)
	}

	if err := s.inscripcionRepo.CancelarInscripcion(ctx, actividadID, usuarioID); err != nil {
		return apperrors.New(500, "no se pudo cancelar la inscripción", err)
	}

	_ = s.actividadRepo.DecrementarInscritos(ctx, actividadID)
	return nil
}

// ─── ObtenerMisInscripciones ──────────────────────────────────────────────────

func (s *Servicio) ObtenerMisInscripciones(ctx context.Context, usuarioID uuid.UUID) ([]*input.MisInscripcionesResponse, error) {
	inscripciones, err := s.inscripcionRepo.ListarPorUsuario(ctx, usuarioID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron obtener las inscripciones", err)
	}

	resp := make([]*input.MisInscripcionesResponse, 0, len(inscripciones))
	for _, ins := range inscripciones {
		act, err := s.actividadRepo.ObtenerPorID(ctx, ins.ActividadID)
		if err != nil || act == nil {
			continue
		}
		resp = append(resp, &input.MisInscripcionesResponse{
			Inscripcion: *toInscripcionResponse(ins),
			Actividad:   *toActividadResponse(act),
		})
	}
	return resp, nil
}

// ─── ListarInscritos ──────────────────────────────────────────────────────────

func (s *Servicio) ListarInscritos(ctx context.Context, actividadID, solicitanteID uuid.UUID) ([]*input.InscripcionActividadResponse, error) {
	act, err := s.actividadRepo.ObtenerPorID(ctx, actividadID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo obtener la actividad", err)
	}
	if act == nil {
		return nil, apperrors.New(404, "actividad no encontrada", nil)
	}

	usuario, err := s.userRepo.FindByIDWithRoles(ctx, solicitanteID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	esAdmin := usuario.HasRole(domain.RoleAdmin) || usuario.HasRole(domain.RoleAdministrativo)
	if act.CreadorID != solicitanteID && !esAdmin {
		return nil, apperrors.New(403, "solo el creador o un administrador puede ver los inscritos", nil)
	}

	inscripciones, err := s.inscripcionRepo.ListarPorActividad(ctx, actividadID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo listar los inscritos", err)
	}

	resp := make([]*input.InscripcionActividadResponse, 0, len(inscripciones))
	for _, ins := range inscripciones {
		resp = append(resp, toInscripcionResponse(ins))
	}
	return resp, nil
}

// ─── Notificaciones async ─────────────────────────────────────────────────────

func (s *Servicio) enviarNotificacionesAsync(act *domain.Actividad) {
	ctx := context.Background()

	terminos := append(act.SkillsRequeridos, act.InteresesRequeridos...)
	if len(terminos) == 0 {
		return
	}

	ids, err := s.userRepo.BuscarIDsPorSkillsOIntereses(ctx, terminos)
	if err != nil {
		return
	}

	for _, uid := range ids {
		if uid == act.CreadorID {
			continue
		}
		n := domain.NewNotification(
			uid,
			domain.NotificationTypeActividadCreada,
			"Nueva actividad: "+act.Titulo,
			"Hay una nueva "+string(act.Tipo)+" que puede interesarte: '"+act.Titulo+"'.",
		)
		_ = s.notificacionRepo.Create(ctx, n)
	}
}

// ─── Helpers de validación ────────────────────────────────────────────────────

func tipoValido(t domain.TipoActividad) bool {
	switch t {
	case domain.TipoActividadCharla,
		domain.TipoActividadAuditoria,
		domain.TipoActividadEvento,
		domain.TipoActividadConvocatoria,
		domain.TipoActividadDeportivo,
		domain.TipoActividadOtro:
		return true
	}
	return false
}

func estadoValido(e domain.EstadoActividad) bool {
	switch e {
	case domain.EstadoActividadPublicada,
		domain.EstadoActividadCancelada,
		domain.EstadoActividadCompletada:
		return true
	}
	return false
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func toActividadResponse(a *domain.Actividad) *input.ActividadResponse {
	skills := a.SkillsRequeridos
	if skills == nil {
		skills = []string{}
	}
	intereses := a.InteresesRequeridos
	if intereses == nil {
		intereses = []string{}
	}
	return &input.ActividadResponse{
		ID:                  a.ID.String(),
		Titulo:              a.Titulo,
		Descripcion:         a.Descripcion,
		Tipo:                string(a.Tipo),
		Estado:              string(a.Estado),
		FechaInicio:         a.FechaInicio.Format(time.RFC3339),
		FechaFin:            input.FormatearFechaOpcional(a.FechaFin),
		Ubicacion:           a.Ubicacion,
		LinkVirtual:         a.LinkVirtual,
		ImagenURL:           a.ImagenURL,
		CapacidadMaxima:     a.CapacidadMaxima,
		CountInscritos:      a.CountInscritos,
		SkillsRequeridos:    skills,
		InteresesRequeridos: intereses,
		CreadorID:           a.CreadorID.String(),
		CreatedAt:           a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           a.UpdatedAt.Format(time.RFC3339),
	}
}

func toInscripcionResponse(ins *domain.InscripcionActividad) *input.InscripcionActividadResponse {
	return &input.InscripcionActividadResponse{
		ID:               ins.ID.String(),
		ActividadID:      ins.ActividadID.String(),
		UsuarioID:        ins.UsuarioID.String(),
		Estado:           string(ins.Estado),
		Modalidad:        ins.Modalidad,
		Observaciones:    ins.Observaciones,
		FechaInscripcion: ins.FechaInscripcion.Format(time.RFC3339),
	}
}
