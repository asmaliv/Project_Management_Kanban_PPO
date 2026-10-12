-- 000002_create_audit_logs.up.sql

CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_role user_role,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id TEXT NOT NULL DEFAULT '',
    project_id UUID,
    before JSONB,
    after JSONB,
    ip TEXT NOT NULL DEFAULT '',
    request_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_audit_logs_occurred_at ON audit_logs (occurred_at);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id);
CREATE INDEX idx_audit_logs_entity_type_id ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_logs_project_id ON audit_logs (project_id);
CREATE INDEX idx_audit_logs_action ON audit_logs (action);

CREATE OR REPLACE FUNCTION audit_logs_block_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_no_update_delete
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_block_mutation();
