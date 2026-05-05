# E2E Checklist - Backend CURN
Checklist rapido para validar integracion con frontend sin tocar base de codigo.
## 1) Migraciones
Ejecutar en este orden:
1. `docs/migrations/20260312_add_user_profile_fields.sql`
2. `docs/migrations/20260312_create_conversations.sql`
3. `docs/migrations/20260312_create_posts_module.sql`
4. `docs/migrations/20260405_create_projects.sql`
5. `docs/migrations/20260405_create_mentorships.sql`
6. `docs/migrations/20260405_create_notifications.sql`
7. `docs/migrations/20260418_add_user_skills_interests.sql`
8. `docs/migrations/20260505_actividades_convocatorias.sql`
## 2) Flujo minimo funcional
1. Registro/Login
2. Actualizacion de perfil (`PUT /api/v1/users/me`)
3. Crear proyecto (`POST /api/v1/projects`)
4. Solicitar mentoria (`POST /api/v1/mentorships/request`)
5. Aceptar/Rechazar desde cuenta mentor
6. Ver notificaciones (`GET /api/v1/notifications`)
7. Marcar notificacion leida (`PATCH /api/v1/notifications/:id/read`)
8. Crear actividad (`POST /api/v1/actividades`)
9. Listar actividades (`GET /api/v1/actividades`)
10. Inscribirse en actividad (`POST /api/v1/actividades/:id/inscribirse`)
11. Ver mis inscripciones (`GET /api/v1/actividades/mis-inscripciones`)
12. Cancelar inscripcion (`DELETE /api/v1/actividades/:id/inscribirse`)
## 3) Criterios de aceptacion
- Los endpoints de proyecto responden 2xx y persisten en DB.
- Mentorias pasan de `pending` a `active` o `rejected`.
- Al decidir mentoria se crea notificacion para el solicitante.
- El endpoint de notificaciones devuelve items ordenados por fecha descendente.
- Crear actividad con `skills_requeridos` genera notificaciones a usuarios con skills coincidentes.
- Inscripcion falla con `409` si ya inscrito o sin cupo.
- Editar/eliminar actividad ajena falla con `403` para no-admin.
- `go test ./...` compila sin errores.
