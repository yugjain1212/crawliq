package models

import (
	"testing"
	"time"
)

func TestCrawl_IsTerminal(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		status CrawlStatus
		want   bool
	}{
		{"pending is non-terminal", CrawlStatusPending, false},
		{"running is non-terminal", CrawlStatusRunning, false},
		{"completed is terminal", CrawlStatusCompleted, true},
		{"failed is terminal", CrawlStatusFailed, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Crawl{Status: tt.status, StartedAt: now}
			if got := c.IsTerminal(); got != tt.want {
				t.Errorf("IsTerminal() = %v, want %v (status=%q)", got, tt.want, tt.status)
			}
		})
	}
}

func TestCrawlStatus_StringValues(t *testing.T) {
	// Guard against accidental string-value changes — these values are
	// persisted in the database and the CHECK constraint relies on
	// them being stable.
	if string(CrawlStatusPending) != "pending" {
		t.Errorf("CrawlStatusPending = %q, want \"pending\"", CrawlStatusPending)
	}
	if string(CrawlStatusRunning) != "running" {
		t.Errorf("CrawlStatusRunning = %q, want \"running\"", CrawlStatusRunning)
	}
	if string(CrawlStatusCompleted) != "completed" {
		t.Errorf("CrawlStatusCompleted = %q, want \"completed\"", CrawlStatusCompleted)
	}
	if string(CrawlStatusFailed) != "failed" {
		t.Errorf("CrawlStatusFailed = %q, want \"failed\"", CrawlStatusFailed)
	}
}