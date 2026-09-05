package workers

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_Run_AllJobsProcessed(t *testing.T) {
	var processed int64
	process := func(job Job, _ time.Duration) Result {
		atomic.AddInt64(&processed, 1)
		return Result{Job: job}
	}

	pool := NewPool(5, 1*time.Second, process)
	jobs := []Job{
		{CrawlID: 1, URL: "https://a"},
		{CrawlID: 1, URL: "https://b"},
		{CrawlID: 1, URL: "https://c"},
		{CrawlID: 1, URL: "https://d"},
		{CrawlID: 1, URL: "https://e"},
		{CrawlID: 1, URL: "https://f"},
		{CrawlID: 1, URL: "https://g"},
		{CrawlID: 1, URL: "https://h"},
	}

	results := pool.Run(context.Background(), jobs)
	if len(results) != len(jobs) {
		t.Errorf("Run() returned %d results, want %d", len(results), len(jobs))
	}
	if got := atomic.LoadInt64(&processed); got != int64(len(jobs)) {
		t.Errorf("processed = %d, want %d", got, len(jobs))
	}
}

func TestPool_Run_EmptyJobs(t *testing.T) {
	var calls int64
	process := func(job Job, _ time.Duration) Result {
		atomic.AddInt64(&calls, 1)
		return Result{Job: job}
	}

	pool := NewPool(3, 1*time.Second, process)
	results := pool.Run(context.Background(), nil)
	if len(results) != 0 {
		t.Errorf("Run(nil) returned %d results, want 0", len(results))
	}
	if atomic.LoadInt64(&calls) != 0 {
		t.Errorf("process called %d times for empty jobs, want 0", atomic.LoadInt64(&calls))
	}
}

func TestPool_Run_ResultSetMatchesJobSet(t *testing.T) {
	process := func(job Job, _ time.Duration) Result {
		return Result{Job: job}
	}

	pool := NewPool(4, 1*time.Second, process)
	in := []Job{
		{CrawlID: 7, URL: "u1"},
		{CrawlID: 7, URL: "u2"},
		{CrawlID: 7, URL: "u3"},
		{CrawlID: 7, URL: "u4"},
	}

	out := pool.Run(context.Background(), in)
	if len(out) != len(in) {
		t.Fatalf("Run() returned %d results, want %d", len(out), len(in))
	}

	// We don't promise order, so collect the URLs we got and verify
	// they match the input set exactly.
	got := make(map[string]bool)
	for _, r := range out {
		got[r.Job.URL] = true
	}
	for _, j := range in {
		if !got[j.URL] {
			t.Errorf("missing result for URL %q", j.URL)
		}
	}
}

func TestPool_Run_NoLeakOnEmptyQueue(t *testing.T) {
	process := func(job Job, _ time.Duration) Result {
		return Result{Job: job}
	}
	pool := NewPool(4, 1*time.Second, process)

	done := make(chan struct{})
	go func() {
		pool.Run(context.Background(), nil)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Run(nil) did not return within 1s — possible goroutine leak")
	}
}

// TestPool_Run_ConcurrentSafety ensures Run() can be called from
// multiple goroutines concurrently without data races (caught by
// -race). Each Run call uses its own channels, so we expect no shared
// state to corrupt.
func TestPool_Run_ConcurrentSafety(t *testing.T) {
	var wg sync.WaitGroup
	var processCalls int64
	process := func(job Job, _ time.Duration) Result {
		atomic.AddInt64(&processCalls, 1)
		return Result{Job: job}
	}

	pool := NewPool(4, 1*time.Second, process)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobs := []Job{{URL: "x"}, {URL: "y"}, {URL: "z"}}
			pool.Run(context.Background(), jobs)
		}()
	}
	wg.Wait()

	want := int64(5 * 3)
	if got := atomic.LoadInt64(&processCalls); got != want {
		t.Errorf("process called %d times, want %d", got, want)
	}
}

func TestResult_Succeeded(t *testing.T) {
	intP := func(v int) *int { return &v }

	// Succeeded() means "we got an HTTP response". Any non-nil
	// status code counts — the actual status (200 vs 404 vs 500) is
	// the caller's responsibility to interpret.
	if !(Result{StatusCode: intP(200)}).Succeeded() {
		t.Error("Result with status 200 should be Succeeded()")
	}
	if !(Result{StatusCode: intP(500)}).Succeeded() {
		t.Error("Result with status 500 should still be Succeeded() (HTTP response was received)")
	}
	if (Result{}).Succeeded() {
		t.Error("Result with no status code should NOT be Succeeded()")
	}
	errStr := "oops"
	if (Result{StatusCode: intP(200), Error: &errStr}).Succeeded() {
		t.Error("Result with non-nil Error should NOT be Succeeded()")
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	var processCalls int64
	process := func(job Job, _ time.Duration) Result {
		atomic.AddInt64(&processCalls, 1)
		return Result{Job: job}
	}

	pool := NewPool(3, 1*time.Second, process)
	dispatcher := NewDispatcher(pool)

	urls := []string{"https://a.example", "https://b.example", "https://c.example"}
	results := dispatcher.Dispatch(context.Background(), 42, urls)

	if len(results) != len(urls) {
		t.Fatalf("Dispatch() returned %d results, want %d", len(results), len(urls))
	}
	if got := atomic.LoadInt64(&processCalls); got != int64(len(urls)) {
		t.Errorf("process called %d times, want %d", got, len(urls))
	}
	for _, r := range results {
		if r.Job.CrawlID != 42 {
			t.Errorf("result CrawlID = %d, want 42", r.Job.CrawlID)
		}
	}
}