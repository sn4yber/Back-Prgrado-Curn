package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config agrupa toda la configuración de la aplicación cargada desde el .env
type Config struct {
	App       AppConfig
	DB        DBConfig
	JWT       JWTConfig
	Argon2    Argon2Config
	RateLimit RateLimitConfig
}

type AppConfig struct {
	Port string
	Env  string
}

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN construye el string de conexión para pgx
func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	KeyLength   uint32
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

// Load carga el .env y devuelve un Config listo para usar.
// Falla rápido si alguna variable crítica no está definida.
func Load() (*Config, error) {
	// En producción el .env puede no existir (variables inyectadas por el SO)
	_ = godotenv.Load()

	cfg := &Config{}

	// ── App ──────────────────────────────────────────────
	cfg.App = AppConfig{
		Port: requireEnv("APP_PORT"),
		Env:  getEnv("APP_ENV", "development"),
	}

	// ── Database ─────────────────────────────────────────
	maxOpen, err := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "25"))
	if err != nil {
		return nil, fmt.Errorf("DB_MAX_OPEN_CONNS inválido: %w", err)
	}
	maxIdle, err := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))
	if err != nil {
		return nil, fmt.Errorf("DB_MAX_IDLE_CONNS inválido: %w", err)
	}
	connLifetime, err := time.ParseDuration(getEnv("DB_CONN_MAX_LIFETIME", "5m"))
	if err != nil {
		return nil, fmt.Errorf("DB_CONN_MAX_LIFETIME inválido: %w", err)
	}

	cfg.DB = DBConfig{
		Host:            requireEnv("DB_HOST"),
		Port:            requireEnv("DB_PORT"),
		User:            requireEnv("DB_USER"),
		Password:        requireEnv("DB_PASSWORD"),
		Name:            requireEnv("DB_NAME"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    maxOpen,
		MaxIdleConns:    maxIdle,
		ConnMaxLifetime: connLifetime,
	}

	// ── JWT ──────────────────────────────────────────────
	accessExpiry, err := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_EXPIRY inválido: %w", err)
	}
	refreshExpiry, err := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_EXPIRY inválido: %w", err)
	}

	cfg.JWT = JWTConfig{
		Secret:        requireEnv("JWT_SECRET"),
		AccessExpiry:  accessExpiry,
		RefreshExpiry: refreshExpiry,
	}

	// ── Argon2id ─────────────────────────────────────────
	memory, err := strconv.ParseUint(getEnv("ARGON2_MEMORY", "65536"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("ARGON2_MEMORY inválido: %w", err)
	}
	iterations, err := strconv.ParseUint(getEnv("ARGON2_ITERATIONS", "3"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("ARGON2_ITERATIONS inválido: %w", err)
	}
	parallelism, err := strconv.ParseUint(getEnv("ARGON2_PARALLELISM", "2"), 10, 8)
	if err != nil {
		return nil, fmt.Errorf("ARGON2_PARALLELISM inválido: %w", err)
	}
	keyLength, err := strconv.ParseUint(getEnv("ARGON2_KEY_LENGTH", "32"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("ARGON2_KEY_LENGTH inválido: %w", err)
	}

	cfg.Argon2 = Argon2Config{
		Memory:      uint32(memory),
		Iterations:  uint32(iterations),
		Parallelism: uint8(parallelism),
		KeyLength:   uint32(keyLength),
	}

	// ── Rate Limiting ─────────────────────────────────────
	rateReqs, err := strconv.Atoi(getEnv("RATE_LIMIT_REQUESTS", "5"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_REQUESTS inválido: %w", err)
	}
	rateWindow, err := time.ParseDuration(getEnv("RATE_LIMIT_WINDOW", "1m"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_WINDOW inválido: %w", err)
	}

	cfg.RateLimit = RateLimitConfig{
		Requests: rateReqs,
		Window:   rateWindow,
	}

	return cfg, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// requireEnv devuelve el valor de la variable o falla con panic si no existe.
// Se usa solo en variables críticas que sin ellas la app no puede arrancar.
func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("variable de entorno requerida no definida: %s", key))
	}
	return val
}

// getEnv devuelve el valor de la variable o el valor por defecto si no existe.
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
