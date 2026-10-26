-- 000005_create_risks.up.sql

CREATE TYPE risk_category AS ENUM (
    'operational',
    'schedule',
    'technical',
    'compliance',
    'financial',
    'security'
);

CREATE TYPE risk_level AS ENUM (
    'low',
    'medium',
    'high',
    'critical'
);

CREATE TYPE risk_status AS ENUM (
    'identified',
    'assessed',
    'mitigating',
    'accepted',
    'closed'
);

CREATE SEQUENCE risk_code_seq START 1;

CREATE TABLE risks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            TEXT NOT NULL UNIQUE,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id         UUID REFERENCES tasks(id) ON DELETE SET NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    category        risk_category NOT NULL,
    likelihood      INT CHECK (likelihood BETWEEN 1 AND 5),
    impact          INT CHECK (impact BETWEEN 1 AND 5),
    score           INT GENERATED ALWAYS AS (likelihood * impact) STORED,
    level           risk_level,
    status          risk_status NOT NULL DEFAULT 'identified',
    mitigation_plan TEXT NOT NULL DEFAULT '',
    owner_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    raised_by       UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at       TIMESTAMPTZ
);

CREATE INDEX idx_risks_project_id_level ON risks (project_id, level);
CREATE INDEX idx_risks_project_id_status ON risks (project_id, status);
CREATE INDEX idx_risks_task_id ON risks (task_id);
CREATE INDEX idx_risks_owner_id ON risks (owner_id);
CREATE INDEX idx_risks_raised_by ON risks (raised_by);
