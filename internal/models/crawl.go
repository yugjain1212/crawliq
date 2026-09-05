package models

import "time"

type CrawlStatus string

const (
	CrawlStatusPending   CrawlStatus = "pending"
	CrawlStatusRunning   CrawlStatus = "running"
	CrawlStatusCompleted CrawlStatus = "completed"
	CrawlStatusFailed    CrawlStatus = "failed"
)

type Crawl struct {
	ID           int64       `json:"id"`
	Website      string      `json:"website"`
	Status       CrawlStatus `json:"status"`
	TotalPages   int         `json:"total_pages"`
	SuccessPages int         `json:"success_pages"`
	FailedPages  int         `json:"failed_pages"`
	StartedAt    time.Time   `json:"started_at"`
	FinishedAt   *time.Time  `json:"finished_at,omitempty"`
}

// IsTerminal reports whether the crawl has finished in any final state
// (either completed successfully or failed). The service / handler layers
// use this to decide whether to keep polling for status updates.
func (c *Crawl) IsTerminal() bool {
	return c.Status == CrawlStatusCompleted || c.Status == CrawlStatusFailed
}