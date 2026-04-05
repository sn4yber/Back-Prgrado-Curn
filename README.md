# CURN Networking Platform - Backend

API REST para la plataforma de networking universitario con enfoque en la reintegracion del egresado.
**Corporacion Universitaria Rafael Nunez**

---

## Stack tecnologico

| Componente | Tecnologia |
|---|---|
| Lenguaje | Go |
| Framework HTTP | Gin |
| Base de datos | PostgreSQL |
| Driver BD | pgx v5 |
| Hash contrasenas | Argon2id |
| Autenticacion | JWT (access + refresh) |
| Logger | Zap |
| Config | godotenv |

## Arquitectura

Arquitectura hexagonal (Ports & Adapters).

- `internal/core/domain`: entidades y reglas de negocio
- `internal/core/ports`: contratos de entrada/salida
- `internal/core/usecases`: casos de uso por modulo
- `internal/adapters/driving/http`: handlers, middlewares y router
- `internal/adapters/driven/persistence/postgres`: repositorios PostgreSQL
- `internal/adapters/driven/storage`: almacenamiento de adjuntos

Referencia: `docs/AUTH.md` y diagramas en `docs/`.

## Modulos implementados (estado actual)

- **Autenticacion**: register, login, refresh token, forgot/reset password
- **Perfil de usuario**: `GET/PUT /api/v1/users/me`
- **Conexiones**: request, accept, reject, block, list
- **Conversaciones**: inbox 1:1 contextual
- **Publicaciones**: base de moderacion institucional y adjuntos (requiere migracion de posts)
- **Comentarios**: comentarios por publicación (create/list)

## Inicio rapido

```bash
# 1) Clonar y entrar

git clone <repo>
cd Back-Prgrado-Curn

# 2) Configurar variables de entorno
cp .env.example .env
# IMPORTANTE: en .env define AUTH_DEFAULT_PROGRAM_ID con un UUID real de programs.

# 3) Ejecutar migraciones necesarias (orden sugerido)
#    Ajusta usuario/DB segun tu entorno.
psql -U postgres -d database-Prgrado -f docs/migrations/20260312_add_user_profile_fields.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260312_create_conversations.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260312_create_posts_module.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260405_create_projects.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260405_create_mentorships.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260405_create_notifications.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260405_seed_programs_and_roles.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260405_create_comments_module.sql

# 4) Levantar API
go run ./cmd/api/main.go
```

## Endpoints disponibles

### Salud

- `GET /health`

### Auth

- `POST /api/v1/auth/register` (name/email/password/program_name/role/document_id)
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`

Contrato de registro (`POST /api/v1/auth/register`):
- `program_name`: nombre del programa (sin UUID)
- `role`: `estudiante` o `egresado`
- `document_id`: cédula obligatoria
- `semester`: obligatorio si `role=estudiante`
- `graduation_date` (`YYYY-MM-DD`): obligatorio si `role=egresado`

### Perfil

- `GET /api/v1/users/me`
- `PUT /api/v1/users/me`

Contrato para frontend (`PUT /api/v1/users/me`):
- Auth: `Bearer <access_token>`
- Body JSON: `name`, `document_id`, `phone`, `city`, `bio`, `student_code`, `semester`, `graduation_year`, `is_graduated`, `linkedin_url`, `github_url`
- Respuesta: objeto completo del perfil persistido (misma estructura de `GET /api/v1/users/me`)

### Conexiones

- `POST /api/v1/connections/request`
- `POST /api/v1/connections/:id/accept`
- `POST /api/v1/connections/:id/reject`
- `POST /api/v1/connections/:id/block`
- `GET /api/v1/connections`

### Conversaciones

- `POST /api/v1/conversations`
- `GET /api/v1/conversations`
- `GET /api/v1/conversations/:id`
- `POST /api/v1/conversations/:id/messages`
- `GET /api/v1/conversations/admin/flagged`

### Publicaciones

- `POST /api/v1/posts`
- `GET /api/v1/posts/mine`
- `GET /api/v1/posts/public`
- `GET /api/v1/posts/pending-review`
- `PATCH /api/v1/posts/:id/moderate`

### Comentarios

- `POST /api/v1/posts/:id/comments`
- `GET /api/v1/posts/:id/comments`

### Moderación (Admin)

- `GET /api/v1/admin/moderation/posts`
- `PATCH /api/v1/admin/moderation/posts/:id`
- `GET /api/v1/admin/moderation/upload-policies`

### Proyectos

- `POST /api/v1/projects`
- `GET /api/v1/projects/mine`
- `GET /api/v1/projects/:id`
- `PATCH /api/v1/projects/:id/status`

### Mentorías

- `POST /api/v1/mentorships/request`
- `POST /api/v1/mentorships/:id/accept`
- `POST /api/v1/mentorships/:id/reject`
- `GET /api/v1/mentorships/mine`

### Notificaciones

- `GET /api/v1/notifications?limit=20&offset=0`
- `PATCH /api/v1/notifications/:id/read`

Contrato para frontend (`POST /api/v1/posts`):
- Auth: `Bearer <access_token>`
- Content-Type: `multipart/form-data`
- Campos requeridos: `title`, `description`, `category`, `originality_declaration`
- Campos opcionales: `declared_author_id`, `coauthor_ids` (CSV), `privacy_consent`, `is_institutional`, `verified_by_faculty`, `attachments` (archivo multiple)
- `category` permitido: `tesis`, `emprendimiento`, `trabajo`

Errores esperados del modulo posts:
- `400`: validacion de entrada (categoria invalida, archivo invalido, extension no permitida, form-data invalido)
- `401`: token invalido o ausente
- `403`: acceso denegado (moderacion admin)
- `404`: publicacion no encontrada
- `500`: error interno no controlado

## Notas de moderacion institucional (posts)

- Valida autoria declarada y coautoria
- Bloquea extensiones de alto riesgo (`.exe`, `.js`, `.py`, etc.)
- Aplica whitelist por categoria (`tesis`, `emprendimiento`, `trabajo`)
- Detecta texto sensible y envia a `pending_review`
- Permite revision administrativa (`admin`/`administrativo`)
- Ofusca datos personales en vista publica cuando no hay consentimiento de privacidad
