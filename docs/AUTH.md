# Módulo de Autenticación — CURN Networking Platform

**Versión:** 1.0  
**Fecha:** 2026-02-19  
**Autor:** Snayber Madrid  
**Proyecto:** Trabajo de Grado — Corporación Universitaria Rafael Núñez

---

## Tabla de contenido

1. [Arquitectura](#arquitectura)
2. [Seguridad](#seguridad)
3. [Configuración](#configuración)
4. [Endpoints](#endpoints)
5. [Flujos de uso](#flujos-de-uso)
6. [Estructura de archivos](#estructura-de-archivos)
7. [Base de datos](#base-de-datos)

---

## Arquitectura

El módulo sigue **arquitectura hexagonal** (Ports & Adapters), lo que garantiza que la lógica de negocio no depende de ningún framework ni base de datos.

```
┌─────────────────────────────────────────────────────────┐
│                   ADAPTADORES PRIMARIOS                  │
│         HTTP Handler → Router → Middleware               │
└───────────────────────┬─────────────────────────────────┘
                        │ llama a
┌───────────────────────▼─────────────────────────────────┐
│                    NÚCLEO (dominio)                      │
│   Ports Input (contratos) → Use Cases → Ports Output    │
└───────────────────────┬─────────────────────────────────┘
                        │ implementado por
┌───────────────────────▼─────────────────────────────────┐
│                  ADAPTADORES SECUNDARIOS                 │
│           PostgreSQL Repositories                        │
└─────────────────────────────────────────────────────────┘
```

**Regla de dependencia:** el dominio nunca importa nada externo. La dependencia siempre apunta hacia adentro.

---

## Seguridad

| Mecanismo | Implementación | Detalle |
|---|---|---|
| Hash de contraseñas | **Argon2id** | Memory: 64MB, Iterations: 3, Parallelism: 2 |
| Access Token | **JWT HS256** | Expiración: 15 minutos |
| Refresh Token | **SHA-256 hash en BD** | Expiración: 7 días, rotación en cada uso |
| Forgot Password Token | **SHA-256 hash en BD** | Expiración: 1 hora, uso único |
| Rate Limiting | **In-memory por IP** | 5 intentos de login por minuto |
| Dominio institucional | **Validación de sufijo** | Solo `@campusuninunez.edu.co` |
| Enumeración de usuarios | **Respuesta silenciosa** | Forgot password siempre devuelve 200 |
| Timing attacks | **Comparación en tiempo constante** | `constantTimeEqual` en verificación argon2 |
| CORS | **Middleware global** | Configurable por entorno |

---

## Configuración

Copia `.env.example` a `.env` y completa los valores:

```env
# Server
APP_PORT=8080
APP_ENV=development          # development | production

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=tu_password
DB_NAME=database-Prgrado
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# JWT
JWT_SECRET=minimo_32_caracteres_aleatorios
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Argon2id
ARGON2_MEMORY=65536
ARGON2_ITERATIONS=3
ARGON2_PARALLELISM=2
ARGON2_KEY_LENGTH=32

# Rate Limiting
RATE_LIMIT_REQUESTS=5
RATE_LIMIT_WINDOW=1m
```

---

## Endpoints

Base URL: `http://localhost:8080`

---

### `GET /health`

Verifica que el servidor está activo.

**Respuesta exitosa `200`**
```json
{ "status": "ok" }
```

---

### `POST /api/v1/auth/register`

Registra un nuevo usuario en la plataforma.

> ⚠️ Solo se aceptan correos con dominio `@campusuninunez.edu.co`

**Body**
```json
{
  "name":       "Snayber Madrid",
  "email":      "smadridi21@campusuninunez.edu.co",
  "password":   "Snayber4567...",
  "program_id": "0a3e24e6-4cbf-4214-a828-10a59e709a24"
}
```

**Respuesta exitosa `201`**
```json
{
  "access_token":  "eyJhbGci...",
  "refresh_token": "BnkzFqhk...",
  "token_type":    "Bearer",
  "expires_in":    900
}
```

**Errores posibles**

| Código | Mensaje |
|---|---|
| `400` | datos de entrada inválidos |
| `400` | solo se permiten correos institucionales (@campusuninunez.edu.co) |
| `409` | el correo ya está registrado |
| `500` | error interno del servidor |

---

### `POST /api/v1/auth/login`

Autentica un usuario existente.

> ⚠️ Rate limit: 5 intentos por IP por minuto.

**Body**
```json
{
  "email":    "smadridi21@campusuninunez.edu.co",
  "password": "Snayber4567..."
}
```

**Respuesta exitosa `200`**
```json
{
  "access_token":  "eyJhbGci...",
  "refresh_token": "yws8Hm2N...",
  "token_type":    "Bearer",
  "expires_in":    900
}
```

**Errores posibles**

| Código | Mensaje |
|---|---|
| `400` | datos de entrada inválidos |
| `401` | credenciales inválidas |
| `403` | cuenta suspendida |
| `403` | cuenta inactiva |
| `429` | demasiados intentos, espera un momento |

---

### `POST /api/v1/auth/refresh`

Genera un nuevo par de tokens usando el refresh token actual.

> El refresh token anterior queda invalidado automáticamente (rotación).

**Body**
```json
{
  "refresh_token": "yws8Hm2N..."
}
```

**Respuesta exitosa `200`**
```json
{
  "access_token":  "eyJhbGci...",
  "refresh_token": "nuevo_token...",
  "token_type":    "Bearer",
  "expires_in":    900
}
```

**Errores posibles**

| Código | Mensaje |
|---|---|
| `400` | refresh_token requerido |
| `401` | refresh token no encontrado |
| `401` | token expirado |

---

### `POST /api/v1/auth/forgot-password`

Solicita un token de recuperación de contraseña.

> Siempre responde `200` aunque el correo no exista — evita enumeración de usuarios.  
> En desarrollo el token se imprime en consola. En producción se enviará por correo.

**Body**
```json
{
  "email": "smadridi21@campusuninunez.edu.co"
}
```

**Respuesta exitosa `200`**
```json
{
  "message": "si el correo está registrado, recibirás instrucciones"
}
```

---

### `POST /api/v1/auth/reset-password`

Restablece la contraseña usando el token recibido.

> El token expira en 1 hora y es de uso único.  
> Al restablecer, todos los refresh tokens activos del usuario quedan invalidados.

**Body**
```json
{
  "token":        "token_recibido_por_correo",
  "new_password": "NuevaPassword123..."
}
```

**Respuesta exitosa `200`**
```json
{
  "message": "contraseña actualizada correctamente"
}
```

**Errores posibles**

| Código | Mensaje |
|---|---|
| `400` | datos de entrada inválidos |
| `400` | token de recuperación inválido o expirado |
| `404` | usuario no encontrado |
| `500` | error interno del servidor |

---

## Flujos de uso

### Flujo de registro y login
```
Cliente                         API
  │                              │
  │── POST /register ──────────► │
  │                              │ valida dominio institucional
  │                              │ hashea password con Argon2id
  │                              │ guarda usuario en BD
  │                              │ genera access + refresh token
  │◄── 201 { tokens } ──────────│
  │                              │
  │── POST /login ─────────────► │
  │                              │ verifica rate limit por IP
  │                              │ verifica credenciales
  │                              │ rota refresh token
  │◄── 200 { tokens } ──────────│
```

### Flujo de recuperación de contraseña
```
Cliente                         API
  │                              │
  │── POST /forgot-password ───► │
  │                              │ genera token seguro (32 bytes)
  │                              │ guarda hash SHA-256 en BD
  │                              │ [TODO] envía token por correo
  │◄── 200 { message } ─────────│
  │                              │
  │── POST /reset-password ────► │
  │                              │ valida token y expiración
  │                              │ hashea nueva password con Argon2id
  │                              │ invalida todos los refresh tokens
  │◄── 200 { message } ─────────│
```

---

## Estructura de archivos

```
internal/
├── core/                          ← Núcleo — sin dependencias externas
│   ├── domain/
│   │   ├── user.go                ← Entidad User, roles, ENUMs, reglas de negocio
│   │   └── auth.go                ← RefreshToken, PasswordResetToken
│   ├── ports/
│   │   ├── input/
│   │   │   └── auth_usecase.go    ← Contratos que el HTTP llama
│   │   └── output/
│   │       └── auth_repository.go ← Contratos que la BD implementa
│   └── usecases/
│       └── auth/
│           └── auth_service.go    ← Toda la lógica de negocio
│
├── adapters/
│   ├── driven/                    ← Adaptadores secundarios (BD)
│   │   └── persistence/
│   │       ├── migrations/
│   │       │   └── 001_auth_tokens.sql
│   │       └── postgres/
│   │           ├── db.go              ← Pool de conexiones pgxpool
│   │           ├── user_repository.go ← Queries de usuarios
│   │           └── auth_repository.go ← Queries de tokens
│   └── driving/                   ← Adaptadores primarios (HTTP)
│       └── http/
│           ├── handler/
│           │   └── auth_handler.go    ← Controllers HTTP
│           ├── middleware/
│           │   └── rate_limiter.go    ← Rate limit + CORS
│           └── router/
│               └── router.go          ← Definición de rutas
│
pkg/
├── config/config.go               ← Variables de entorno
├── errors/errors.go               ← Errores de dominio tipados
└── logger/logger.go               ← Logger estructurado (zap)

cmd/
└── api/
    └── main.go                    ← Punto de entrada y ensamblado
```

---

## Base de datos

### Tablas nuevas creadas (migración `001_auth_tokens.sql`)

**`refresh_tokens`**

| Columna | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users |
| `token_hash` | TEXT | SHA-256 del token, nunca el token plano |
| `expires_at` | TIMESTAMPTZ | Expiración: 7 días |
| `created_at` | TIMESTAMPTZ | Fecha de creación |

**`password_reset_tokens`**

| Columna | Tipo | Descripción |
|---|---|---|
| `id` | UUID | PK |
| `user_id` | UUID | FK → users |
| `token_hash` | TEXT | SHA-256 del token |
| `expires_at` | TIMESTAMPTZ | Expiración: 1 hora |
| `used` | BOOLEAN | true = ya fue utilizado |
| `created_at` | TIMESTAMPTZ | Fecha de creación |

---

*Documentación generada el 2026-02-19 — CURN Networking Platform v1.0*

