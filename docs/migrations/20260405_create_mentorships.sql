-- Modulo de mentorias entre estudiantes y egresados.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'mentorship_status') THEN
        CREATE TYPE mentorship_status AS ENUM ('pending', 'active', 'rejected');
    END IF;
END$$;
CREATE TABLE IF NOT EXISTS mentorships (
    id UUID PRIMARY KEY,
    requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mentor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status mentorship_status NOT NULL DEFAULT 'pending',
    request_message TEXT,
    decision_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    responded_at TIMESTAMPTZ,
    CONSTRAINT mentorships_no_self_request CHECK (requester_id <> mentor_id)
);
CREATE INDEX IF NOT EXISTS idx_mentorships_requester_id ON mentorships(requester_id);
CREATE INDEX IF NOT EXISTS idx_mentorships_mentor_id ON mentorships(mentor_id);
CREATE INDEX IF NOT EXISTS idx_mentorships_project_id ON mentorships(project_id);
CREATE INDEX IF NOT EXISTS idx_mentorships_status ON mentorships(status);
