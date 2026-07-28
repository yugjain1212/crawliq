package workers

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Pool struct {
	workerCount int
	timeout     time.Duration
	process     ProcessFunc
}

func NewPool(workerCount int, timeout time.Duration, process ProcessFunc) *Pool {
	return &Pool{
		workerCount: workerCount,
		timeout:     timeout,
		process:     process,
	}
}

func (p *Pool) Run(ctx context.Context, jobs []Job) []Result {
	jobCh := make(chan Job, len(jobs))
	resultCh := make(chan Result, len(jobs))

	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	var wg sync.WaitGroup
	wg.Add(p.workerCount)

	for i := 0; i < p.workerCount; i++ {
		workerID := i
		go p.worker(ctx, workerID, jobCh, resultCh, &wg)
	}

	// Close resultCh once every worker has exited, so the range loop
	// below terminates instead of blocking forever.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make([]Result, 0, len(jobs))
	for r := range resultCh {
		results = append(results, r)
	}

	return results
}
func (p *Pool) worker(ctx context.Context, id int, jobCh <-chan Job, resultCh chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Debug().Int("worker_id", id).Msg("worker stopping: context cancelled")
			return

		case job, ok := <-jobCh:
			if !ok {
				// jobCh closed and drained — no more work, this worker
				// can exit cleanly.
				return
			}

			result := p.process(job, p.timeout)
			resultCh <- result
		}
	}
}
