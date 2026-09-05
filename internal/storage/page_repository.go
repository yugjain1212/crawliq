package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yugjain1212/crawliq/internal/models"
)

// ErrPageNotFound is the sentinel returned when a single-page lookup
// can't find a row, mirroring ErrCrawlNotFound's role for the crawls
// table. Handlers turn it into a 404.
var ErrPageNotFound = errors.New("page not found")

type PageRepository struct {
	db *pgxpool.Pool
}

func NewPageRepository(db *pgxpool.Pool) *PageRepository {
	return &PageRepository{db: db}
}

// Create inserts a single page row. The (crawl_id, url) unique index
// means ON CONFLICT DO NOTHING is the right behaviour: if the same URL
// appears twice in a sitemap (which happens), we silently keep the
// first record rather than failing the whole insert.
func (r *PageRepository) Create(ctx context.Context, p *models.Page) error {
	const query = `
		INSERT INTO pages
		    (crawl_id, url, status_code, content_type, content_length, response_time_ms, title, html, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (crawl_id, url) DO NOTHING
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query,
		p.CrawlID, p.URL, p.StatusCode, p.ContentType,
		p.ContentLength, p.ResponseTimeMs, p.Title, p.HTML, p.Error,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		// ON CONFLICT DO NOTHING with RETURNING returns no rows when
		// the conflict is hit, which surfaces here as pgx.ErrNoRows.
		// Treat that as a non-error: the row was simply already
		// present from a previous insert.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("inserting page %s: %w", p.URL, err)
	}
	return nil
}

// BulkInsert uses pgx's CopyFrom protocol to write many page rows in
// one round-trip. This is dramatically faster than per-row INSERTs for
// crawls that produce thousands of pages.
func (r *PageRepository) BulkInsert(ctx context.Context, pages []*models.Page) (int64, error) {
	if len(pages) == 0 {
		return 0, nil
	}

	rows := make([][]interface{}, len(pages))
	for i, p := range pages {
		rows[i] = []interface{}{
			p.CrawlID, p.URL, p.StatusCode, p.ContentType,
			p.ContentLength, p.ResponseTimeMs, p.Title, p.HTML, p.Error,
		}
	}

	copyCount, err := r.db.CopyFrom(
		ctx,
		pgx.Identifier{"pages"},
		[]string{
			"crawl_id", "url", "status_code", "content_type",
			"content_length", "response_time_ms", "title", "html", "error",
		},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return 0, fmt.Errorf("bulk inserting %d pages: %w", len(pages), err)
	}
	return copyCount, nil
}

// GetByID fetches a single page row by primary key.
func (r *PageRepository) GetByID(ctx context.Context, id int64) (*models.Page, error) {
	const query = `
		SELECT id, crawl_id, url, status_code, content_type, content_length,
		       response_time_ms, title, html, error, created_at
		FROM pages
		WHERE id = $1
	`

	var p models.Page
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.CrawlID, &p.URL, &p.StatusCode, &p.ContentType, &p.ContentLength,
		&p.ResponseTimeMs, &p.Title, &p.HTML, &p.Error, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPageNotFound
		}
		return nil, fmt.Errorf("fetching page %d: %w", id, err)
	}
	return &p, nil
}

// PageListOptions controls pagination for GetByCrawlID. Crawls can have
// thousands of pages, so returning all of them unpaginated by default
// would be a real performance and payload-size problem.
type PageListOptions struct {
	Limit  int
	Offset int
}

// GetByCrawlID fetches a page of results belonging to a single crawl,
// ordered by insertion order (oldest first). Also returns the total row
// count so callers (the API handler) can build proper pagination
// metadata for the client, not just the current page of results.
func (r *PageRepository) GetByCrawlID(ctx context.Context, crawlID int64, opts PageListOptions) ([]*models.Page, int64, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	const countQuery = `SELECT COUNT(*) FROM pages WHERE crawl_id = $1`
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, crawlID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting pages for crawl %d: %w", crawlID, err)
	}

	const query = `
		SELECT id, crawl_id, url, status_code, content_type, content_length,
		       response_time_ms, title, html, error, created_at
		FROM pages
		WHERE crawl_id = $1
		ORDER BY id ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, crawlID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("fetching pages for crawl %d: %w", crawlID, err)
	}
	defer rows.Close()

	pages := make([]*models.Page, 0, opts.Limit)
	for rows.Next() {
		var p models.Page
		if err := rows.Scan(
			&p.ID, &p.CrawlID, &p.URL, &p.StatusCode, &p.ContentType, &p.ContentLength,
			&p.ResponseTimeMs, &p.Title, &p.HTML, &p.Error, &p.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning page row: %w", err)
		}
		pages = append(pages, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating pages for crawl %d: %w", crawlID, err)
	}

	return pages, total, nil
}