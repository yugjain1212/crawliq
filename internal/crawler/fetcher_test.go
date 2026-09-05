package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetcher_Fetch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><head><title>Hello</title></head><body></body></html>"))
	}))
	defer server.Close()

	f := NewFetcher(5*time.Second, 5, "CrawlIQ-Test/1.0")
	res, err := f.Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Fetch() unexpected error: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Fetch() status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if !strings.Contains(res.ContentType, "text/html") {
		t.Errorf("Fetch() content-type = %q, want to contain text/html", res.ContentType)
	}
	if !strings.Contains(string(res.Body), "Hello") {
		t.Errorf("Fetch() body = %q, want to contain \"Hello\"", string(res.Body))
	}
	if res.ResponseTime <= 0 {
		t.Errorf("Fetch() response time = %v, want > 0", res.ResponseTime)
	}
	if res.ContentLength != int64(len(res.Body)) {
		t.Errorf("Fetch() content length = %d, want %d", res.ContentLength, len(res.Body))
	}
}

func TestFetcher_Fetch_404IsStillAStatus(t *testing.T) {
	// A 404 should come back with StatusCode=404 and NO error — the
	// fetch itself succeeded, we just got an unsuccessful status.
	// The error path is for fetch-layer failures (DNS, TCP, etc.).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	f := NewFetcher(5*time.Second, 5, "test")
	res, err := f.Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Fetch() unexpected error for 404: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Fetch() status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

func TestFetcher_Fetch_BadURL(t *testing.T) {
	f := NewFetcher(1*time.Second, 5, "test")
	_, err := f.Fetch(context.Background(), "://bad-url")
	if err == nil {
		t.Error("Fetch() expected error for malformed URL, got nil")
	}
}

func TestFetcher_Fetch_ContextCancellation(t *testing.T) {
	// Build a server that hangs forever so we can prove the request
	// respects context cancellation.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	f := NewFetcher(30*time.Second, 5, "test")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := f.Fetch(ctx, server.URL)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Fetch() expected error when context is cancelled, got nil")
	}
	// Sanity: should bail out well before the 30s client timeout.
	if elapsed > 5*time.Second {
		t.Errorf("Fetch() took %v despite context cancellation — should have bailed early", elapsed)
	}
}

func TestFetcher_RedirectLimit(t *testing.T) {
	// Server that always redirects to itself — this should be stopped
	// by the configured redirect limit, not loop forever.
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Redirect(w, r, r.URL.String(), http.StatusFound)
	}))
	defer server.Close()

	f := NewFetcher(2*time.Second, 3, "test")
	_, err := f.Fetch(context.Background(), server.URL)
	if err == nil {
		t.Error("Fetch() expected redirect-limit error, got nil")
	}
	if hits > 10 {
		t.Errorf("Fetch() followed too many redirects before bailing: %d", hits)
	}
}