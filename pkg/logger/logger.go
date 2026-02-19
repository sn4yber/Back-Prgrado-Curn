package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger es el contrato que expone la aplicación.
// Ningún paquete importa zap directamente — solo este contrato.
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Audit(msg string, fields ...zap.Field)
}

// zapLogger es la implementación interna basada en zap.
type zapLogger struct {
	log   *zap.Logger
	audit *zap.Logger
}

// New construye un Logger según el entorno recibido.
// "development" → logs legibles en consola.
// cualquier otro valor → logs JSON para producción.
func New(env string) (Logger, error) {
	appLog, err := buildAppLogger(env)
	if err != nil {
		return nil, err
	}

	auditLog, err := buildAuditLogger()
	if err != nil {
		return nil, err
	}

	return &zapLogger{
		log:   appLog,
		audit: auditLog,
	}, nil
}

// ─── Implementación de la interfaz ───────────────────────────────────────────

func (z *zapLogger) Info(msg string, fields ...zap.Field) {
	z.log.Info(msg, fields...)
}

func (z *zapLogger) Warn(msg string, fields ...zap.Field) {
	z.log.Warn(msg, fields...)
}

func (z *zapLogger) Error(msg string, fields ...zap.Field) {
	z.log.Error(msg, fields...)
}

func (z *zapLogger) Fatal(msg string, fields ...zap.Field) {
	z.log.Fatal(msg, fields...)
}

// Audit escribe en el logger de auditoría de seguridad.
// Úsalo para: login, logout, registro, cambios de contraseña, rate limit excedido.
func (z *zapLogger) Audit(msg string, fields ...zap.Field) {
	z.audit.Info(msg, fields...)
}

// ─── Constructores internos ───────────────────────────────────────────────────

func buildAppLogger(env string) (*zap.Logger, error) {
	if env == "development" {
		return buildDevelopmentLogger()
	}
	return buildProductionLogger()
}

// buildDevelopmentLogger produce logs coloreados y legibles en consola.
func buildDevelopmentLogger() (*zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return cfg.Build()
}

// buildProductionLogger produce logs JSON estructurados para monitoreo.
func buildProductionLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}

// buildAuditLogger siempre produce JSON estructurado independiente del entorno.
// En el futuro puede redirigirse a un archivo o servicio externo.
func buildAuditLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.InitialFields = map[string]interface{}{
		"log_type": "audit",
	}
	return cfg.Build()
}
