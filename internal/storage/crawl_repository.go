package storage

import

var ErrCrawlNotFound = errors.New("crawl not found")

type CrawlRepository struct {
	db *pgxpool.Pool
}
