ALTER TABLE messages
ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL,
ADD COLUMN IF NOT EXISTS deleted_by UUID NULL;

CREATE TABLE IF NOT EXISTS message_events (
    id UUID PRIMARY KEY,
    message_id UUID NOT NULL,
    conversation_id UUID NOT NULL,
    actor_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    old_content TEXT NULL,
    new_content TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_message_events_message_id ON message_events (message_id);
CREATE INDEX IF NOT EXISTS idx_message_events_conversation_id ON message_events (conversation_id);
CREATE INDEX IF NOT EXISTS idx_message_events_created_at ON message_events (created_at DESC);

