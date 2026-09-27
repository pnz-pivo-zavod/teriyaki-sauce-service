-- +goose Up
CREATE TABLE tasks (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT      NOT NULL,
    name         TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    date         TIMESTAMPTZ,
    notify_at    TIMESTAMPTZ,
    priority     SMALLINT    NOT NULL DEFAULT 0 CHECK (priority BETWEEN 0 AND 3),
    is_completed BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX tasks_user_id_date_idx ON tasks (user_id, date);

CREATE TABLE tags (
    id      BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name    TEXT   NOT NULL,
    color   TEXT   NOT NULL DEFAULT '',
    UNIQUE (user_id, name)
);

CREATE TABLE task_tags (
    task_id BIGINT NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    tag_id  BIGINT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)
);

CREATE INDEX task_tags_tag_id_idx ON task_tags (tag_id);

CREATE TABLE notes (
    id      BIGSERIAL PRIMARY KEY,
    task_id BIGINT      NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    text    TEXT        NOT NULL,
    date    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX notes_task_id_idx ON notes (task_id);

-- +goose Down
DROP TABLE notes, task_tags, tags, tasks;
