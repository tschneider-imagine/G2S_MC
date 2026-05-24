package store

const phase2FMessageSendResultMigration = `
ALTER TABLE message_journal ADD COLUMN http_status_code INTEGER;
ALTER TABLE message_journal ADD COLUMN latency_ms INTEGER;
ALTER TABLE message_journal ADD COLUMN response_excerpt TEXT;
ALTER TABLE message_journal ADD COLUMN sent_at DATETIME;
ALTER TABLE message_journal ADD COLUMN completed_at DATETIME;
ALTER TABLE message_journal ADD COLUMN transport_mode TEXT;
`
