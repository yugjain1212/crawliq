package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yugjain1212/crawliq/internal/models"
)

var ErrCrawlNotFound = errors.New("crawl not found")

type CrawlRepository struct {
	db *pgxpool.Pool
}

func NewCrawlRepository(db *pgxpool.Pool) *CrawlRepository {
	return &CrawlRepository{db: db}
}

func (r *CrawlRepository) Create(ctx context.Context, wesbsite string) (*models.Crawl, error) {
	const query = `
	INSERT INTO crawls (website, status)
	VALUES ($1, $2)
	RETURNING id, website, status, total_pages, success_page, failed_pages, started_at, finished_at
	`
	var c models.Crawl
	err := r.db.QueryRow(ctx, query, website, models.CrawlStatusPending).Scan(
		&c.ID, &c.Website, &c.Status, &c.TotalPages, &c.SuccessPages, &c.FailedPages, &c.StartedAt, &c.FinishedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting crawl :%w", err)

	}
	return &c, nil
}

func (r *CrawlRepository) GetByID(ctx context.Context, id int64) (*models.Crawl, error) {
	const query = `
	SELECT id, website, status, total_pages, success_pages, failed_pages, started_at, finished_at
	FROM crawls
	WHERE id = $1
    `
	var c models.Crawl
	err := r.db.QueryRow(ctx, query, id).Scan(&c.ID, &c.Website, &c.Status, &c.TotalPages, &c.SuccessPages, &c.FailedPages, &c.StartedAt, &c.FinishedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCrawlNotFound
		}
		return nil, fmt.Errorf("Fetching crawl %d: %w", id, err)
	}
	return &c, nil
}

func (r *CrawlRepository) UpdateStatus(ctx context.Context, id int64, status models.CrawlStatus) error {
	const query = `UPDATE crawls SET status = $1 WHERE id = $2`
	tag, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("Upadating Crawl %d status %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCrawlNotFound
	}
	return nil

}

type CompleteStats struct {
	Status      models.CrawlStatus
	TotalPages  int
	Success     int
	FailedPages int
}

func (r *CrawlRepository) complete(ctx context.Context, id int64, stats CompleteStats) error {
	const query = `
	UPDATE crawls
	SET status = $1,
	total_pages = $2,
	success_pages = $3,
	failed_pages = $4,
	finished_at = now()
	WHERE id = $5
	`
	tag, err := r.db.Exec(ctx, query, stats.Status, stats.TotalPages, stats.SuccessPages, stats.FailedPages, id)
	if err != nil {
		return fmt.Errorf("completing crawl %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCrawlNotFound
	}
	return nil

}

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
