package models

import "time"

type Page struct {
	ID             int64     `json:"id"`
	CrawlID        int64     `json:"crawl_id"`
	URL            string    `json:"url"`
	StatusCode     *int      `json:"status_code"`
	ContentType    *string   `json:"content_type"`
	ContentLength  *int64    `json:"content_length"`
	ResponseTimeMs *int      `json:"response_time_ms"`
	Title          *string   `json:"title"`
	HTML           *string   `json:"html,omitempty"`
	Error          *string   `json:"error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (p *Page) Succeeded() bool {
	return p.StatusCode != nil && *p.StatusCode >= 200 && *p.StatusCode < 400

}
