-- Migración para asegurar que se puedan adjuntar múltiples archivos por publicación
-- Elimina la restricción única en la columna post_id si existe.

DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM pg_constraint 
        WHERE conname = 'post_attachments_post_id_key' 
          AND conrelid = 'post_attachments'::regclass
    ) THEN
        ALTER TABLE post_attachments DROP CONSTRAINT post_attachments_post_id_key;
    END IF;

    -- También revisamos si existe un índice único en post_id que no sea constraint y lo eliminamos
    IF EXISTS (
        SELECT 1
        FROM pg_class c
        JOIN pg_index i ON c.oid = i.indexrelid
        WHERE c.relname = 'post_attachments_post_id_idx'
          AND i.indisunique = true
    ) THEN
        DROP INDEX post_attachments_post_id_idx;
        CREATE INDEX post_attachments_post_id_idx ON post_attachments(post_id);
    END IF;
END $$;
