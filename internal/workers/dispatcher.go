package workers

import "context"

type Dispatcher struct {
	pool *Pool
}

func NewDispatcher(pool *Pool) *Dispatcher {
	return &Dispatcher{
		pool: pool,
	}
}

func (d *Dispatcher) Dispatcher(ctx context.Context, crawlID int64, urls []string) []Result {
	jobs := make([]Job, len(urls))
	for i, u := range urls {
		jobs[i] = Job{
			CrawlID: crawlID,
			URL:     u,
		}
	}

	return d.pool.Run(ctx, jobs)

}
