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

## Inicio rapido

```bash
# 1) Clonar y entrar

git clone <repo>
cd Back-Prgrado-Curn

# 2) Configurar variables de entorno
cp .env.example .env

# 3) Ejecutar migraciones necesarias (orden sugerido)
#    Ajusta usuario/DB segun tu entorno.
psql -U postgres -d database-Prgrado -f docs/migrations/20260312_add_user_profile_fields.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260312_create_conversations.sql
psql -U postgres -d database-Prgrado -f docs/migrations/20260312_create_posts_module.sql

# 4) Levantar API
go run ./cmd/api/main.go
```

## Endpoints disponibles

### Salud

- `GET /health`

### Auth

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`

### Perfil

- `GET /api/v1/users/me`
- `PUT /api/v1/users/me`

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

## Notas de moderacion institucional (posts)

- Valida autoria declarada y coautoria
- Bloquea extensiones de alto riesgo (`.exe`, `.js`, `.py`, etc.)
- Aplica whitelist por categoria (`tesis`, `emprendimiento`, `trabajo`)
- Detecta texto sensible y envia a `pending_review`
- Permite revision administrativa (`admin`/`administrativo`)
- Ofusca datos personales en vista publica cuando no hay consentimiento de privacidad
