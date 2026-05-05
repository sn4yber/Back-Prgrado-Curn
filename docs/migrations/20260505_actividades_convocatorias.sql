-- Tipos enumerados para actividades
DO $$ BEGIN
    CREATE TYPE tipo_actividad AS ENUM ('charla','auditoria','evento','convocatoria','deportivo','otro');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE estado_actividad AS ENUM ('publicada','cancelada','completada');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE estado_inscripcion_actividad AS ENUM ('inscrito','cancelado','asistido');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Tabla principal de actividades y convocatorias
CREATE TABLE IF NOT EXISTS actividades (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo               TEXT NOT NULL,
    descripcion          TEXT NOT NULL,
    tipo                 tipo_actividad NOT NULL,
    estado               estado_actividad NOT NULL DEFAULT 'publicada',

    fecha_inicio         TIMESTAMPTZ NOT NULL,
    fecha_fin            TIMESTAMPTZ,
    ubicacion            TEXT,
    link_virtual         TEXT,
    imagen_url           TEXT,

    capacidad_maxima     INT,
    count_inscritos      INT NOT NULL DEFAULT 0,

    skills_requeridos    TEXT[] NOT NULL DEFAULT '{}',
    intereses_requeridos TEXT[] NOT NULL DEFAULT '{}',

    creador_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_actividades_tipo    ON actividades(tipo);
CREATE INDEX IF NOT EXISTS idx_actividades_estado  ON actividades(estado);
CREATE INDEX IF NOT EXISTS idx_actividades_creador ON actividades(creador_id);
CREATE INDEX IF NOT EXISTS idx_actividades_fecha   ON actividades(fecha_inicio);

-- Tabla de inscripciones
CREATE TABLE IF NOT EXISTS inscripciones_actividades (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actividad_id      UUID NOT NULL REFERENCES actividades(id) ON DELETE CASCADE,
    usuario_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    estado            estado_inscripcion_actividad NOT NULL DEFAULT 'inscrito',
    fecha_inscripcion TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(actividad_id, usuario_id)
);

CREATE INDEX IF NOT EXISTS idx_inscripciones_actividad ON inscripciones_actividades(actividad_id);
CREATE INDEX IF NOT EXISTS idx_inscripciones_usuario   ON inscripciones_actividades(usuario_id);
