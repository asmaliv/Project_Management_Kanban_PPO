-- 000003_create_projects_members_columns.up.sql

CREATE TYPE project_status AS ENUM ('active', 'on_hold', 'completed', 'archived');

CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status      project_status NOT NULL DEFAULT 'active',
    owner_id    UUID NOT NULL REFERENCES users(id),
    start_date  DATE,
    end_date    DATE,
    created_by  UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_projects_owner_id  ON projects (owner_id);
CREATE INDEX idx_projects_status    ON projects (status);
CREATE INDEX idx_projects_deleted_at ON projects (deleted_at) WHERE deleted_at IS NULL;

-- Sequence terpisah untuk project code (PRJ-0001, PRJ-0002, ...)
CREATE SEQUENCE project_code_seq START 1;

CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    added_by   UUID NOT NULL REFERENCES users(id),
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX idx_project_members_user_id ON project_members (user_id);

CREATE TABLE board_columns (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    position       INTEGER NOT NULL,
    is_done_column BOOLEAN NOT NULL DEFAULT FALSE,
    wip_limit      INTEGER,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_columns_position_unique UNIQUE (project_id, position) DEFERRABLE INITIALLY DEFERRED,
    CONSTRAINT board_columns_wip_limit_positive CHECK (wip_limit IS NULL OR wip_limit > 0)
);

-- Tepat satu kolom Done per proyek
CREATE UNIQUE INDEX idx_board_columns_one_done_per_project
    ON board_columns (project_id)
    WHERE is_done_column = TRUE;

CREATE INDEX idx_board_columns_project_id ON board_columns (project_id, position);
