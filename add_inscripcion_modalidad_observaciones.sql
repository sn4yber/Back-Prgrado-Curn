ALTER TABLE inscripciones_actividades
    ADD COLUMN IF NOT EXISTS modalidad    VARCHAR(20),
    ADD COLUMN IF NOT EXISTS observaciones TEXT;
