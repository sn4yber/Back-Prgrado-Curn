-- Agrega campos de perfil personal/academico al esquema actual.
-- Ejecutar una sola vez en PostgreSQL antes de usar PUT /api/v1/users/me.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS document_id VARCHAR(20),
    ADD COLUMN IF NOT EXISTS phone VARCHAR(20),
    ADD COLUMN IF NOT EXISTS city VARCHAR(100),
    ADD COLUMN IF NOT EXISTS student_code VARCHAR(20),
    ADD COLUMN IF NOT EXISTS semester SMALLINT,
    ADD COLUMN IF NOT EXISTS graduation_year SMALLINT,
    ADD COLUMN IF NOT EXISTS is_graduated BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS linkedin_url TEXT,
    ADD COLUMN IF NOT EXISTS github_url TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_document_id_unique
    ON users (document_id)
    WHERE document_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_graduation_year
    ON users (graduation_year);

CREATE INDEX IF NOT EXISTS idx_users_is_graduated
    ON users (is_graduated);

