package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxRedirects = 10

type FetchResult struct {
	StatusCode    int
	ContentType   string
	ContentLength int64
	ResponseTime  time.Duration
	Body          []byte
	FinalURL      string
}

type Fetcher struct {
	client    *http.Client
	userAgent string
}

func NewFetcher(timeout time.Duration, userAgent string) *Fetcher {
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

	return &FetchResult{
		StatusCode:    resp.StatusCode,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: int64(len(body)),
		ResponseTime:  elapsed,
		Body:          body,
		FinalURL:      resp.Request.URL.String(),
	}, nil
}
