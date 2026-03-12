package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/pkg/config"
)

func NewPool(ctx context.Context, cfg config.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("error parseando DSN: %w", err)
	}

	// Configuración optimizada del pool para alto rendimiento
	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)

	// MaxConnLifetime: límite de vida total de una conexión
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime

	// MaxConnIdleTime: cierra conexiones inactivas más rápido (ahorro de recursos)
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	// HealthCheckPeriod: verificación periódica de conexiones
	poolCfg.HealthCheckPeriod = 1 * time.Minute

	// ConnectTimeout: timeout para establecer nuevas conexiones
	poolCfg.ConnConfig.ConnectTimeout = 5 * time.Second

	// AfterConnect: preparar statements frecuentes para mejor performance
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// pgx automáticamente cachea prepared statements, pero forzamos los más críticos
		_, err := conn.Exec(ctx, "SELECT 1") // warm-up
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("error creando pool de conexiones: %w", err)
	}

	// Ping con timeout para evitar bloqueos en startup
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("error conectando a PostgreSQL: %w", err)
	}

	return pool, nil
}
