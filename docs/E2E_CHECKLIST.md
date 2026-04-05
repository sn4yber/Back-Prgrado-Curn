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
## 2) Flujo minimo funcional
1. Registro/Login
2. Actualizacion de perfil (`PUT /api/v1/users/me`)
3. Crear proyecto (`POST /api/v1/projects`)
4. Solicitar mentoria (`POST /api/v1/mentorships/request`)
5. Aceptar/Rechazar desde cuenta mentor
6. Ver notificaciones (`GET /api/v1/notifications`)
7. Marcar notificacion leida (`PATCH /api/v1/notifications/:id/read`)
## 3) Criterios de aceptacion
- Los endpoints de proyecto responden 2xx y persisten en DB.
- Mentorias pasan de `pending` a `active` o `rejected`.
- Al decidir mentoria se crea notificacion para el solicitante.
- El endpoint de notificaciones devuelve items ordenados por fecha descendente.
- `go test ./...` compila sin errores.
