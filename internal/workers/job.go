package workers

import "time"

// Job represents a single unit of crawl work: the crawl it belongs to
// and the URL the worker should fetch + parse.
type Job struct {
	CrawlID int64
	URL     string
}

// Result carries everything the worker learned about a single page back
// to the dispatcher / service layer. Nullable fields are pointers so a
// failed fetch (no status code, no title) is distinguishable from a
// successful fetch with a zero value.
type Result struct {
	Job            Job
	StatusCode     *int
	ContentType    *string
	ContentLength  *int64
	ResponseTimeMs *int
	Title          *string
	HTML           *string
	Error          *string
}

// Succeeded reports whether the worker actually managed to fetch the
// page and got an HTTP response (the actual status code is still
// inspected by callers — 404 still "succeeded" as a fetch).
//
// Defined on a value receiver so callers can use either Result or
// *Result interchangeably without a dereference.
func (r Result) Succeeded() bool {
	return r.Error == nil && r.StatusCode != nil
}

// ProcessFunc is the worker pool's pluggable per-job handler — typically
// the crawler's ProcessJob method. Kept as a function type so the pool
// stays decoupled from the crawler package.
type ProcessFunc func(job Job, timeout time.Duration) Result