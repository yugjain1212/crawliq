package workers

import "context"

// Dispatcher is the thin layer between a slice of URLs and the worker
// pool. It exists so callers (the crawl service) don't need to know how
// to construct Job values themselves, and so we have one obvious place
// to add cross-cutting concerns later (rate limiting, prioritization,
// per-host fairness, etc.) without touching the pool.
type Dispatcher struct {
	pool *Pool
}

func NewDispatcher(pool *Pool) *Dispatcher {
	return &Dispatcher{pool: pool}
}

// Dispatch converts the given URLs into Jobs and runs them through the
// worker pool, returning the full set of results (one per URL, in
// completion order — not necessarily input order).
func (d *Dispatcher) Dispatch(ctx context.Context, crawlID int64, urls []string) []Result {
	jobs := make([]Job, len(urls))
	for i, u := range urls {
		jobs[i] = Job{
			CrawlID: crawlID,
			URL:     u,
		}
	}

	return d.pool.Run(ctx, jobs)
}