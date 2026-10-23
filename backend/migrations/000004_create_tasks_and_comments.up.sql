-- 000004_create_tasks_and_comments.up.sql

CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'critical');

CREATE TABLE tasks (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    column_id        UUID NOT NULL REFERENCES board_columns(id) ON DELETE RESTRICT,
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    priority         task_priority NOT NULL DEFAULT 'medium',
    assignee_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    reporter_id      UUID NOT NULL REFERENCES users(id),
    due_date         DATE,
    progress_percent INTEGER NOT NULL DEFAULT 0,
    position         INTEGER NOT NULL DEFAULT 0,
    labels           TEXT[] NOT NULL DEFAULT '{}',
    version          INTEGER NOT NULL DEFAULT 1,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ,
    CONSTRAINT tasks_progress_percent_check CHECK (progress_percent >= 0 AND progress_percent <= 100),
    CONSTRAINT tasks_version_check CHECK (version >= 1)
);

CREATE INDEX idx_tasks_project_col_pos ON tasks (project_id, column_id, position) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_assignee_id     ON tasks (assignee_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_due_date        ON tasks (due_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_deleted_at      ON tasks (deleted_at);

CREATE TABLE task_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id    UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_comments_task_created ON task_comments (task_id, created_at ASC);
CREATE INDEX idx_task_comments_author       ON task_comments (author_id);
