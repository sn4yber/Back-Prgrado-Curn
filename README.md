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

## Guia rapida para frontend

### URL base y autenticacion

- Base local: `http://localhost:8080`
- Header para rutas protegidas:
  - `Authorization: Bearer <access_token>`
- Acceso sin token:
  - `GET /health`
  - `POST /api/v1/auth/*`
  - `GET /uploads/*` (archivos publicados)

### Flujo recomendado en frontend

1. `register` (solo cuando el usuario se crea por primera vez).
2. `login` y guardar `access_token` + `refresh_token`.
3. Consumir rutas protegidas con `Bearer <access_token>`.
4. Si una llamada responde `401`, invocar `POST /api/v1/auth/refresh` y reintentar.
5. Si `refresh` falla, cerrar sesion y redirigir a login.

### Ejemplo de login + refresh (contrato)

`POST /api/v1/auth/login`

```json
{
  "email": "smadridi21@campusuninunez.edu.co",
  "password": "snayber4589##"
}
```

Respuesta exitosa:

```json
{
  "access_token": "<jwt>",
  "refresh_token": "<token>",
  "token_type": "Bearer",
  "expires_in": 900
}
```

`POST /api/v1/auth/refresh`

```json
{
  "refresh_token": "<refresh_token>"
}
```

### Matriz de endpoints para consumo

> Todos los endpoints bajo `/api/v1` son protegidos por JWT, excepto `/api/v1/auth/*`.

#### Salud

- `GET /health`

#### Auth

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/admin/curn/register`
- `POST /api/v1/auth/admin/curn/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`

Contrato de registro (`POST /api/v1/auth/register`):
- Requeridos: `name`, `email`, `password`, `program_name`, `role`, `document_id`
- `role`: `estudiante` o `egresado`
- Si `role=estudiante`: `semester` obligatorio
- Si `role=egresado`: `graduation_date` (`YYYY-MM-DD`) obligatorio

Reglas de negocio del registro (importante para frontend):
- `program_id` ya no se envia desde frontend; el backend resuelve el programa por `program_name`.
- `program_name` debe existir en tabla `programs` (cargada con `docs/migrations/20260405_seed_programs_and_roles.sql`).
- La busqueda de `program_name` se hace de forma tolerante a mayusculas/minusculas y acentos.
- `role` permitido: solo `estudiante` o `egresado`.
- El backend asigna el rol en `user_roles` automaticamente por nombre de rol.
- `document_id` es obligatorio y unico por usuario.
- `email` debe ser institucional: `@campusuninunez.edu.co`.

Contrato de registro admin CURN (`POST /api/v1/auth/admin/curn/register`):
- Requeridos: `name`, `email`, `password`, `document_id`
- `email` obligatorio con dominio `@curn.edu.co`
- `program_name` opcional: si no se envía, el backend usa `AUTH_DEFAULT_PROGRAM_ID`
- El backend asigna rol `admin` automáticamente

Contrato de login admin CURN (`POST /api/v1/auth/admin/curn/login`):
- Requeridos: `email`, `password`
- `email` obligatorio con dominio `@curn.edu.co`
- Solo autentica usuarios con rol `admin` o `administrativo`

Ejemplo estudiante:

```json
{
  "name": "Estudiante Demo",
  "email": "estudiante.demo@campusuninunez.edu.co",
  "password": "Demo1234##",
  "program_name": "Ingenieria de Sistemas",
  "role": "estudiante",
  "document_id": "1234567890",
  "semester": 8
}
```

Ejemplo egresado:

```json
{
  "name": "Egresado Demo",
  "email": "egresado.demo@campusuninunez.edu.co",
  "password": "Demo1234##",
  "program_name": "Ingenieria de Sistemas",
  "role": "egresado",
  "document_id": "1098765432",
  "graduation_date": "2024-12-15"
}
```

#### Perfil

- `GET /api/v1/users/me`
- `PUT /api/v1/users/me`
- `GET /api/v1/users/:id/public`

Body permitido en `PUT /api/v1/users/me` (patch parcial):
- `name`, `document_id`, `phone`, `city`, `bio`
- `student_code`, `semester`, `graduation_year`, `is_graduated`
- `linkedin_url`, `github_url`

#### Conexiones

- `POST /api/v1/connections/request`
- `POST /api/v1/connections/:id/accept`
- `POST /api/v1/connections/:id/reject`
- `POST /api/v1/connections/:id/block`
- `GET /api/v1/connections`

Payload minimo para request:

```json
{
  "addressee_id": "<uuid>"
}
```

#### Conversaciones

- `POST /api/v1/conversations`
- `GET /api/v1/conversations`
- `GET /api/v1/conversations/:id`
- `POST /api/v1/conversations/:id/messages`
- `GET /api/v1/conversations/admin/flagged` (rol admin/administrativo)

Payload minimo para iniciar:

```json
{
  "other_user_id": "<uuid>",
  "source_type": "post",
  "source_id": "<post_id>",
  "first_message": "Me interesa tu publicacion"
}
```

Reglas institucionales clave:
- Solo se permite `source_type=post`
- Solo se puede iniciar contra el autor del post
- No se permite self-chat

#### Publicaciones

- `POST /api/v1/posts` (multipart/form-data)
- `GET /api/v1/posts/mine`
- `GET /api/v1/posts/public`
- `GET /api/v1/posts/pending-review`
- `PATCH /api/v1/posts/:id/moderate`

Contrato para `POST /api/v1/posts`:
- Requeridos: `title`, `description`, `category`, `originality_declaration`
- Opcionales: `declared_author_id`, `coauthor_ids` (CSV), `privacy_consent`, `is_institutional`, `verified_by_faculty`, `attachments`
- `category` permitido: `tesis`, `emprendimiento`, `trabajo`

#### Comentarios

- `POST /api/v1/posts/:id/comments`
- `GET /api/v1/posts/:id/comments`

Payload minimo para crear:

```json
{
  "content": "Excelente publicacion"
}
```

#### Moderacion (Admin)

- `GET /api/v1/admin/moderation/posts`
- `PATCH /api/v1/admin/moderation/posts/:id`
- `GET /api/v1/admin/moderation/upload-policies`

#### Proyectos

- `POST /api/v1/projects`
- `GET /api/v1/projects/mine`
- `GET /api/v1/projects/:id`
- `PATCH /api/v1/projects/:id/status`

#### Mentorias

- `POST /api/v1/mentorships/request`
- `POST /api/v1/mentorships/:id/accept`
- `POST /api/v1/mentorships/:id/reject`
- `GET /api/v1/mentorships/mine`

#### Notificaciones

- `GET /api/v1/notifications?limit=20&offset=0`
- `PATCH /api/v1/notifications/:id/read`

### Errores comunes para frontend

- `400`: payload invalido o formato incorrecto
- `401`: token invalido/expirado o ausente
- `403`: sin permisos por rol o regla de negocio
- `404`: recurso no encontrado
- `409`: conflicto (correo o cedula ya registrados)
- `422`: regla de dominio no cumplida
- `500`: error interno

Formato de error comun:

```json
{
  "error": "mensaje"
}
```

### Checklist de conexion frontend-backend

- Login obtiene `access_token` y `refresh_token`.
- Requests protegidos incluyen `Authorization: Bearer <access_token>`.
- Interceptor HTTP implementa refresh automatico en `401`.
- Logout limpia tokens locales y estado de usuario.
- UI maneja mensajes de error en base a `error` del backend.

## Notas de moderacion institucional (posts)

- Valida autoria declarada y coautoria
- Bloquea extensiones de alto riesgo (`.exe`, `.js`, `.py`, etc.)
- Aplica whitelist por categoria (`tesis`, `emprendimiento`, `trabajo`)
- Detecta texto sensible y envia a `pending_review`
- Permite revision administrativa (`admin`/`administrativo`)
- Ofusca datos personales en vista publica cuando no hay consentimiento de privacidad
