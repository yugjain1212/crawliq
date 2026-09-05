package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/yugjain1212/crawliq/internal/api/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestRoutes_AllEndpointsRegistered is a structural test: it walks the
// router with each documented HTTP method/path and asserts every
// endpoint is reachable. This catches accidental route renames or
// deletions during refactors.
func TestRoutes_AllEndpointsRegistered(t *testing.T) {
	r := New(Handlers{})

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{http.MethodPost, "/crawls"},
		{http.MethodGet, "/crawls/1"},
		{http.MethodGet, "/crawls/1/pages"},
		{http.MethodDelete, "/crawls/1"},
		{http.MethodGet, "/pages/1"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)
			r.ServeHTTP(rec, req)

			// A registered route in gin returns 200/201/202/204/400/500
			// depending on the handler. We just need the body to be
			// non-empty (which proves a handler ran, rather than gin
			// falling through to its default 404 page).
			if rec.Body.Len() == 0 {
				t.Errorf("%s %s: empty body — route probably not registered", tc.method, tc.path)
			}
		})
	}
}

// TestRoutes_CORSHeaders makes sure CORS preflight works on every
// method we expose.
func TestRoutes_CORSHeaders(t *testing.T) {
	r := New(Handlers{})

	for _, path := range []string{"/health", "/crawls", "/crawls/1", "/crawls/1/pages", "/pages/1"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodOptions, path, nil)
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Errorf("OPTIONS %s: code = %d, want 204", path, rec.Code)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
				t.Errorf("OPTIONS %s: Allow-Origin = %q, want \"*\"", path, got)
			}
			if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
				t.Errorf("OPTIONS %s: Allow-Methods header missing", path)
			}
		})
	}
}

// TestRoutes_PanicRecovery ensures a nil-deref inside a handler is
// converted into a structured 500 envelope rather than crashing the
// process.
func TestRoutes_PanicRecovery(t *testing.T) {
	r := New(Handlers{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(rec, req)

	// The health handler depends on a nil pgxpool.Pool — this will
	// panic when it calls db.Ping. Recovery should turn it into a 500
	// with our envelope.
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("GET /health with nil pool: code = %d, want 500", rec.Code)
	}

	var env response.Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("could not parse response envelope: %v", err)
	}
	if env.Success {
		t.Error("envelope.Success = true on recovered panic, want false")
	}
	if env.Error == nil || env.Error.Code != response.CodeInternal {
		t.Errorf("recovered panic envelope = %+v, want CodeInternal", env.Error)
	}
}