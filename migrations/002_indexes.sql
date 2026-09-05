-- +goose Up
-- +goose StatementBegin
-- Index for listing all pages of a crawl sorted by insertion order
-- (already covered implicitly by the UNIQUE index on (crawl_id, url),
-- but an explicit btree on crawl_id alone speeds up the COUNT(*)
-- query in PageRepository.GetByCrawlID).
CREATE INDEX IF NOT EXISTS idx_pages_crawl_id ON pages (crawl_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Index for filtering crawls by status (handy for the future
-- monitoring/dashboard view; not used by Phase 1 endpoints but cheap
-- to add while we're here).
CREATE INDEX IF NOT EXISTS idx_crawls_status ON crawls (status);
-- +goose StatementEnd

-- +goose StatementBegin
-- Index on started_at for "most recent crawls" listing.
CREATE INDEX IF NOT EXISTS idx_crawls_started_at ON crawls (started_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_crawls_started_at;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_crawls_status;
-- +goose StatementEnd
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_pages_crawl_id;
-- +goose StatementEnd