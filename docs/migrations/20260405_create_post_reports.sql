-- Reportes comunitarios de publicaciones para moderacion post-publicacion.

CREATE TABLE IF NOT EXISTS post_reports (
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    reporter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, reporter_id)
);

CREATE INDEX IF NOT EXISTS idx_post_reports_post_id ON post_reports(post_id);
CREATE INDEX IF NOT EXISTS idx_post_reports_reporter_id ON post_reports(reporter_id);
CREATE INDEX IF NOT EXISTS idx_post_reports_created_at ON post_reports(created_at DESC);

