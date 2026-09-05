package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// hardMaxRedirects is a safety ceiling regardless of what the user
// configures, so a malicious server can't loop a worker forever via
// 100k redirect responses.
const hardMaxRedirects = 50

// FetchResult is the raw output of a single HTTP fetch. The body is
// kept in full because the parser downstream wants the HTML, not just
// a status code.
type FetchResult struct {
	StatusCode    int
	ContentType   string
	ContentLength int64
	ResponseTime  time.Duration
	Body          []byte
	FinalURL      string
}

// Fetcher wraps an *http.Client with our timeout / user-agent /
// redirect policy. One Fetcher is shared by every worker in the pool
// (http.Client is safe for concurrent use) and there is no per-call
// allocation, so a busy pool doesn't churn the heap.
type Fetcher struct {
	client    *http.Client
	userAgent string
}

// NewFetcher builds a Fetcher with the given per-request timeout,
// configured maximum redirect count, and User-Agent header. A negative
// maxRedirects is clamped to the safety ceiling rather than disabling
// redirects entirely — disabling would let a "redirect to itself"
// server hang a worker, which is exactly what this guard is for.
func NewFetcher(timeout time.Duration, maxRedirects int, userAgent string) *Fetcher {
	if maxRedirects <= 0 || maxRedirects > hardMaxRedirects {
		maxRedirects = hardMaxRedirects
	}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		},
	}

	return &Fetcher{
		client:    client,
		userAgent: userAgent,
	}
}

// Fetch issues a GET for the given URL, returns the response body
// along with timing/size metadata, and is bounded by both the
// per-request context and the underlying http.Client.Timeout.
//
// Any error returned here is a *fetch* failure (DNS, TCP, TLS, redirect
// limit, read error, etc.) — the caller is responsible for recording
// it on the worker Result so the page row reflects the failure.
func (f *Fetcher) Fetch(ctx context.Context, targetURL string) (*FetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", f.userAgent)

	start := time.Now()
	resp, err := f.client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	finalURL := targetURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	return &FetchResult{
		StatusCode:    resp.StatusCode,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: int64(len(body)),
		ResponseTime:  elapsed,
		Body:          body,
		FinalURL:      finalURL,
	}, nil
}