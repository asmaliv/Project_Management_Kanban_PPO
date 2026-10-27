-- 000006_create_audit_findings.up.sql

CREATE TYPE finding_severity AS ENUM (
    'low',
    'medium',
    'high',
    'critical'
);

CREATE TYPE finding_status AS ENUM (
    'open',
    'responded',
    'closed'
);

CREATE SEQUENCE audit_finding_code_seq START 1;

CREATE TABLE audit_findings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            TEXT NOT NULL UNIQUE,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id         UUID REFERENCES tasks(id) ON DELETE SET NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    severity        finding_severity NOT NULL,
    status          finding_status NOT NULL DEFAULT 'open',
    raised_by       UUID NOT NULL REFERENCES users(id),
    response        TEXT NOT NULL DEFAULT '',
    responded_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    responded_at    TIMESTAMPTZ,
    closed_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    closed_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_findings_project_id_status ON audit_findings (project_id, status);
CREATE INDEX idx_audit_findings_project_id_severity ON audit_findings (project_id, severity);
CREATE INDEX idx_audit_findings_task_id ON audit_findings (task_id);
CREATE INDEX idx_audit_findings_raised_by ON audit_findings (raised_by);
CREATE INDEX idx_audit_findings_status ON audit_findings (status);
