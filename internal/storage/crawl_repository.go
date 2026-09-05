package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yugjain1212/crawliq/internal/models"
)

// ErrCrawlNotFound is returned by repository methods when the requested
// crawl row does not exist. Handlers translate this to HTTP 404 instead
// of the generic 500, which is the whole point of the sentinel-error
// pattern: distinguishing "not found" from "the database is broken".
var ErrCrawlNotFound = errors.New("crawl not found")

type CrawlRepository struct {
	db *pgxpool.Pool
}

func NewCrawlRepository(db *pgxpool.Pool) *CrawlRepository {
	return &CrawlRepository{db: db}
}

// Create inserts a new crawl row in the pending state and returns the
// fully-populated model (including DB-generated id and started_at).
func (r *CrawlRepository) Create(ctx context.Context, website string) (*models.Crawl, error) {
	const query = `
		INSERT INTO crawls (website, status)
		VALUES ($1, $2)
		RETURNING id, website, status, total_pages, success_pages, failed_pages, started_at, finished_at
	`
	var c models.Crawl
	err := r.db.QueryRow(ctx, query, website, models.CrawlStatusPending).Scan(
		&c.ID, &c.Website, &c.Status, &c.TotalPages,
		&c.SuccessPages, &c.FailedPages, &c.StartedAt, &c.FinishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting crawl: %w", err)
	}
	return &c, nil
}

// GetByID fetches a single crawl row or returns ErrCrawlNotFound.
func (r *CrawlRepository) GetByID(ctx context.Context, id int64) (*models.Crawl, error) {
	const query = `
		SELECT id, website, status, total_pages, success_pages, failed_pages, started_at, finished_at
		FROM crawls
		WHERE id = $1
	`
	var c models.Crawl
	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Website, &c.Status, &c.TotalPages,
		&c.SuccessPages, &c.FailedPages, &c.StartedAt, &c.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCrawlNotFound
		}
		return nil, fmt.Errorf("fetching crawl %d: %w", id, err)
	}
	return &c, nil
}

// UpdateStatus moves a crawl to the given status. Returns
// ErrCrawlNotFound if the id doesn't exist (0 rows affected).
func (r *CrawlRepository) UpdateStatus(ctx context.Context, id int64, status models.CrawlStatus) error {
	const query = `UPDATE crawls SET status = $1 WHERE id = $2`
	tag, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("updating crawl %d status: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCrawlNotFound
	}
	return nil
}

// CompleteStats is the terminal snapshot we write to a crawl row once
// the worker pool has drained.
type CompleteStats struct {
	Status      models.CrawlStatus
	TotalPages  int
	Success     int
	FailedPages int
}

// Complete finalizes a crawl: stamps finished_at = now() and writes the
// final tally of pages attempted / succeeded / failed.
func (r *CrawlRepository) Complete(ctx context.Context, id int64, stats CompleteStats) error {
	const query = `
		UPDATE crawls
		SET status       = $1,
		    total_pages  = $2,
		    success_pages = $3,
		    failed_pages = $4,
		    finished_at  = now()
		WHERE id = $5
	`
	tag, err := r.db.Exec(ctx, query, stats.Status, stats.TotalPages, stats.Success, stats.FailedPages, id)
	if err != nil {
		return fmt.Errorf("completing crawl %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCrawlNotFound
	}
	return nil
}

// Delete removes a crawl row. Pages are removed by ON DELETE CASCADE
// (see migrations), so callers don't have to clean them up themselves.
func (r *CrawlRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM crawls WHERE id = $1`
	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting crawl %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCrawlNotFound
	}
	return nil
}