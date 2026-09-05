-- +goose Up
-- +goose StatementBegin
CREATE TABLE crawls (
    id              BIGSERIAL PRIMARY KEY,
    website         TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'pending',
    total_pages     INTEGER     NOT NULL DEFAULT 0,
    success_pages   INTEGER     NOT NULL DEFAULT 0,
    failed_pages    INTEGER     NOT NULL DEFAULT 0,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at     TIMESTAMPTZ,

    CONSTRAINT crawls_status_chk CHECK (
        status IN ('pending', 'running', 'completed', 'failed')
    )
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE pages (
    id                BIGSERIAL PRIMARY KEY,
    crawl_id          BIGINT      NOT NULL REFERENCES crawls(id) ON DELETE CASCADE,
    url               TEXT        NOT NULL,
    status_code       INTEGER,
    content_type      TEXT,
    content_length    BIGINT,
    response_time_ms  INTEGER,
    title             TEXT,
    html              TEXT,
    error             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT pages_url_per_crawl_unique UNIQUE (crawl_id, url)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS pages;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS crawls;
-- +goose StatementEnd