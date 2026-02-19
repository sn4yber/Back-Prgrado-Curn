# CURN Networking Platform — Backend

API REST para la plataforma de networking universitario con enfoque en la reintegración del egresado.  
**Corporación Universitaria Rafael Núñez**

---

## Stack tecnológico

| Componente | Tecnología |
|---|---|
| Lenguaje | Go 1.23 |
| Framework HTTP | Gin |
| Base de datos | PostgreSQL |
| Driver BD | pgx v5 |
| Hash contraseñas | Argon2id |
| Autenticación | JWT (golang-jwt/jwt v5) |
| Logger | Zap (Uber) |
| Config | godotenv |

## Arquitectura

Arquitectura hexagonal (Ports & Adapters) — ver [`docs/AUTH.md`](docs/AUTH.md)

## Inicio rápido

```bash
# 1. Clonar y entrar al proyecto
git clone <repo>
cd Back-Prgrado-Curn

# 2. Configurar variables de entorno
cp .env.example .env
# editar .env con tus credenciales

# 3. Ejecutar migraciones en PostgreSQL
psql -U postgres -d database-Prgrado -f internal/adapters/driven/persistence/migrations/001_auth_tokens.sql

# 4. Arrancar el servidor
go run ./cmd/api/main.go
```

## Documentación de módulos

| Módulo | Documento |
|---|---|
| Autenticación | [`docs/AUTH.md`](docs/AUTH.md) |

## Endpoints disponibles

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/health` | Health check |
| `POST` | `/api/v1/auth/register` | Registro de usuario |
| `POST` | `/api/v1/auth/login` | Login |
| `POST` | `/api/v1/auth/refresh` | Renovar tokens |
| `POST` | `/api/v1/auth/forgot-password` | Solicitar recuperación |
| `POST` | `/api/v1/auth/reset-password` | Restablecer contraseña |

