-- Modulo de publicaciones con moderacion institucional y adjuntos.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'post_category') THEN
        CREATE TYPE post_category AS ENUM ('tesis', 'emprendimiento', 'trabajo');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'post_status') THEN
        CREATE TYPE post_status AS ENUM ('published', 'flagged', 'pending_review', 'shadow_banned', 'rejected');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    declared_author_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    coauthor_ids UUID[] NOT NULL DEFAULT '{}',

    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    category post_category NOT NULL,

    originality_declaration BOOLEAN NOT NULL,
    privacy_consent BOOLEAN NOT NULL,
    is_institutional BOOLEAN NOT NULL DEFAULT FALSE,
    verified_by_faculty BOOLEAN NOT NULL DEFAULT FALSE,

    status post_status NOT NULL DEFAULT 'published',
    moderation_notes TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts(author_id);
CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);
CREATE INDEX IF NOT EXISTS idx_posts_category ON posts(category);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);

CREATE TABLE IF NOT EXISTS post_attachments (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_url TEXT NOT NULL,
    file_ext VARCHAR(10) NOT NULL,
    mime_type VARCHAR(120) NOT NULL,
    size_bytes BIGINT NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_post_attachments_post_id ON post_attachments(post_id);

