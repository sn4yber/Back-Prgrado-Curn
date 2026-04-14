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
psql -U postgres -d database-Prgrado -f docs/migrations/20260405_create_post_reactions.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260405_create_post_reports.sql

# 4) Levantar API
go run ./cmd/api/main.go
```

## Guia rapida para frontend

## Docker (API + PostgreSQL)

Este repositorio ahora incluye:

- `Dockerfile` para construir la API.
- `docker-compose.yml` para levantar API y PostgreSQL vinculados.
- `docker/postgres/init/00_extensions.sql` para habilitar `pgcrypto` en inicialización de DB.

En este compose, la base de datos no se publica hacia Internet: PostgreSQL queda accesible solo por red interna Docker (`api` <-> `db`).

Pasos mínimos:

```bash
cp .env.example .env
# Ajusta DB_PASSWORD y JWT_SECRET antes de subir servicios

docker compose up -d --build
docker compose logs -f api
```

Recomendación VPS (DigitalOcean): abre solo `8080` (o mejor `80/443` si pones reverse proxy) y no abras `5432` en firewall.

Detener servicios:

```bash
docker compose down
```

Recrear desde cero (elimina datos del volumen de PostgreSQL):

```bash
docker compose down -v
docker compose up -d --build
```

> Nota: la carpeta `docs/migrations/` está vacía en el estado actual del repo, así que no hay migraciones versionadas para aplicar automáticamente en Docker. El archivo `docs/backup` puede servir como base, pero conviene alinear su esquema con las tablas/columnas que usa hoy el código antes de usarlo como inicialización oficial.

## Optimización aplicada (menos recursos, mejor respuesta)

Cambios aplicados para un VPS:

- El healthcheck del contenedor API usa `GET /live` (liveness liviano) en lugar de `GET /health`.
- `GET /health` mantiene validación real de dependencia DB (readiness).
- El pool de PostgreSQL ya no precalienta conexiones mínimas (`MinConns=0`) y cierra ociosas más rápido.
- Se redujeron defaults de conexiones DB para no sobredimensionar en VPS pequeños.
- La consulta pública de posts se optimizó usando agregaciones por `JOIN` en vez de subconsultas correlacionadas por fila.
- Se limitaron logs de contenedores (`max-size`, `max-file`) para evitar crecimiento de disco.

Beneficios esperados:

- Menor uso de RAM/CPU en DB y API cuando hay poco tráfico.
- Menor carga de DB causada por healthchecks internos.
- Mejor escalabilidad de `GET /api/v1/posts/public` en listados grandes.
- Menos riesgo de caída por disco lleno en producción.

## Producción: Nginx + HTTPS + API + DB

Archivos agregados:

- `docker-compose.prod.yml`
- `deploy/nginx/conf.d/app.conf`
- `scripts/backup_postgres.sh`

Beneficios de este esquema:

- Nginx termina TLS y expone solo `80/443` al público.
- La API y PostgreSQL quedan en red interna Docker.
- Gzip en Nginx reduce payload y mejora TTFB percibido en cliente.
- Renovación automática de certificados con `certbot renew`.
- Backups automáticos de DB con retención local y subida opcional a Spaces.

### Levante productivo (referencia)

1) Ajusta dominio/certificados en `deploy/nginx/conf.d/app.conf` reemplazando `example.com`.

2) Construye y etiqueta imagen API:

```bash
docker build -t curn-api:latest .
```

3) Levanta stack prod:

```bash
docker compose -f docker-compose.prod.yml up -d
docker compose -f docker-compose.prod.yml logs -f nginx
```

### Backup automático PostgreSQL

Ejecución manual:

```bash
./scripts/backup_postgres.sh
```

Programación diaria (02:30):

```bash
crontab -l > /tmp/current_cron || true
echo "30 2 * * * cd /ruta/Back-Prgrado-Curn && ./scripts/backup_postgres.sh >> /var/log/curn_backup.log 2>&1" >> /tmp/current_cron
crontab /tmp/current_cron
rm -f /tmp/current_cron
```

Para subir a DigitalOcean Spaces, exporta:

- `UPLOAD_TO_SPACES=true`
- `SPACES_BUCKET`
- `SPACES_REGION`
- `SPACES_ENDPOINT` (ej. `https://nyc3.digitaloceanspaces.com`)
- credenciales AWS compatibles (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)

### CORS para frontend (local y producción)

La API ahora controla orígenes permitidos por variable de entorno:

- `CORS_ALLOWED_ORIGINS` (CSV)

Ejemplos:

```bash
# Local
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173

# Producción
CORS_ALLOWED_ORIGINS=https://app.tudominio.com,https://admin.tudominio.com
```

Si el frontend se consume desde dominio distinto al backend, agrega ese dominio aquí y reinicia el servicio para aplicar cambios.

### URL base y autenticacion

- Base local: `http://localhost:8080`
- Header para rutas protegidas:
  - `Authorization: Bearer <access_token>`
- Acceso sin token:
  - `GET /health`
  - `POST /api/v1/auth/*`
  - `GET /api/v1/catalog/faculties-programs`
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

`GET /api/v1/users/me` devuelve campos espejo para autocompletar frontend:
- `role`, `document_id`, `program_id`, `program_name`, `faculty_name`, `semester`, `graduation_date`, `is_graduated`

#### Catálogo académico (público)

- `GET /api/v1/catalog/faculties-programs`

Respuesta agrupada por facultad con sus programas para el formulario de registro.

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
- `PUT /api/v1/posts/:id`
- `DELETE /api/v1/posts/:id`
- `GET /api/v1/posts/mine`
- `GET /api/v1/posts/public`
- `POST /api/v1/posts/:id/reactions`
- `POST /api/v1/posts/:id/reports`
- `GET /api/v1/posts/pending-review`
- `PATCH /api/v1/posts/:id/moderate`

Contrato para `POST /api/v1/posts`:
- Requeridos: `title`, `description`, `category`, `originality_declaration`
- Opcionales: `declared_author_id`, `coauthor_ids` (CSV), `privacy_consent`, `is_institutional`, `verified_by_faculty`, `attachments`, `is_job_offer`
- `category` permitido: `tesis`, `emprendimiento`, `trabajo`

Reglas institucionales activas en publicación:
- `tesis`: requiere `attachments` (PDF) y metadatos `faculty`, `academic_program`, `advisor`
- `is_job_offer=true`: solo permitido para roles `egresado`, `admin` o `administrativo`
- contenido con lenguaje grave/fraude académico: bloqueo (`422`)
- contenido con términos comerciales ambiguos o datos sensibles: se publica y queda con nota de monitoreo para admin

Contrato para `PUT /api/v1/posts/:id`:
- Body: `{ "title": "...", "description": "...", "category": "tesis|emprendimiento|trabajo" }`
- Regla: solo el autor de la publicación puede editar

Contrato para `DELETE /api/v1/posts/:id`:
- Regla: solo el autor de la publicación puede eliminar

Contrato para `POST /api/v1/posts/:id/reactions`:
- Body: `{ "type": "like" | "love" | "dislike" }`
- Regla: si el usuario ya reaccionó, se actualiza su reacción (upsert)

Contrato para `POST /api/v1/posts/:id/reports`:
- Body: `{ "reason": "fraude | plagio | ofensivo | datos_personales | otro" }`
- Regla: 1 reporte por usuario por publicación
- Auto-moderación: al llegar a 3 reportes, el backend oculta la publicación (`shadow_banned`) y la deja para revisión admin

Semáforo de moderación en respuestas:
- `moderation_level=verde`: publicación visible sin alertas
- `moderation_level=amarillo`: publicación visible con alerta de revisión
- `moderation_level=rojo`: publicación bloqueada/oculta (`shadow_banned` o `rejected`)

Trazabilidad normativa:
- `rule_code`: código interno de regla activada (ej. `RULE_HABEAS_DATA_WARNING`)
- `rule_message`: mensaje legible asociado a la regla

Respuesta de `GET /api/v1/posts/public` incluye:
- `author_name` con el nombre del autor de la publicación
- `reactions_summary` con conteos agregados (`likes`, `love`, `dislike`)
- `current_user_reaction` con la reacción del usuario autenticado o `null`
- `likes_count` con total de reacciones del post
- `comments_count` con total de comentarios del post

#### Comentarios

- `POST /api/v1/posts/:id/comments`
- `GET /api/v1/posts/:id/comments`

Payload minimo para crear:

```json
{
  "content": "Excelente publicacion"
}

`GET /api/v1/posts/:id/comments` incluye `author_name` en cada comentario.
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
