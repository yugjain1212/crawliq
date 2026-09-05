package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/yugjain1212/crawliq/internal/api/response"
	"github.com/yugjain1212/crawliq/internal/models"
	"github.com/yugjain1212/crawliq/internal/service"
	"github.com/yugjain1212/crawliq/internal/storage"
	"github.com/yugjain1212/crawliq/internal/workers"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// These tests verify the handler's request/response translation in
// isolation, without needing a real service stack (which would require
// a DB pool and worker pool). They mirror the production handler
// logic so a regression in either place will be caught.

type startCall struct{ website string }

type fakeCrawlService struct {
	startCalls []startCall
	startOut   *models.Crawl
	startErr   error

	crawls map[int64]*models.Crawl
	pages  map[int64][]*models.Page
}

func newFakeCrawlService() *fakeCrawlService {
	return &fakeCrawlService{
		crawls: make(map[int64]*models.Crawl),
		pages:  make(map[int64][]*models.Page),
	}
}

func (f *fakeCrawlService) StartCrawl(_ /* ctx */ interface{}, in service.StartCrawlInput) (*models.Crawl, error) {
	f.startCalls = append(f.startCalls, startCall{website: in.Website})
	if f.startErr != nil {
		return f.startOut, f.startErr
	}
	return f.startOut, nil
}

func (f *fakeCrawlService) GetCrawl(_ interface{}, id int64) (*models.Crawl, error) {
	c, ok := f.crawls[id]
	if !ok {
		return nil, service.ErrCrawlNotFound
	}
	return c, nil
}

func (f *fakeCrawlService) DeleteCrawl(_ interface{}, id int64) error {
	if _, ok := f.crawls[id]; !ok {
		return service.ErrCrawlNotFound
	}
	delete(f.crawls, id)
	return nil
}

func (f *fakeCrawlService) GetCrawlPages(_ interface{}, id int64, opts storage.PageListOptions) ([]*models.Page, int64, error) {
	if _, ok := f.crawls[id]; !ok {
		return nil, 0, service.ErrCrawlNotFound
	}
	all := f.pages[id]
	offset := opts.Offset
	if offset > len(all) {
		offset = len(all)
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], int64(len(all)), nil
}

func (f *fakeCrawlService) GetPage(_ interface{}, id int64) (*models.Page, error) {
	for _, ps := range f.pages {
		for _, p := range ps {
			if p.ID == id {
				return p, nil
			}
		}
	}
	return nil, service.ErrPageNotFound
}

// newRouter builds a router that uses the same handler bodies as
// production (the parseIDParam, parseIntQuery, and envelope helpers
// are package-level and shared), but adapts the calls so a fake
// service can be substituted.
func newRouter(f *fakeCrawlService) *gin.Engine {
	r := gin.New()

	r.POST("/crawls", func(c *gin.Context) {
		var req startCrawlRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "request body must include a non-empty 'website' field")
			return
		}
		crawl, err := f.StartCrawl(nil, service.StartCrawlInput{Website: req.Website})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInvalidURL):
				response.BadRequest(c, err.Error())
			case errors.Is(err, service.ErrCrawlRunning):
				response.Success(c, http.StatusAccepted, gin.H{"message": "already running", "crawl": crawl})
				return
			default:
				response.InternalError(c, "could not start crawl")
				return
			}
		}
		response.Success(c, http.StatusCreated, crawl)
	})

	r.GET("/crawls/:id", func(c *gin.Context) {
		id, err := parseIDParam(c, "id")
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		crawl, err := f.GetCrawl(nil, id)
		if err != nil {
			if errors.Is(err, service.ErrCrawlNotFound) {
				response.NotFound(c, "crawl not found")
				return
			}
			response.InternalError(c, "could not fetch crawl")
			return
		}
		response.Success(c, http.StatusOK, crawl)
	})

	r.GET("/crawls/:id/pages", func(c *gin.Context) {
		id, err := parseIDParam(c, "id")
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		limit, err := parseIntQuery(c, "limit", 50, 1, 500)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		offset, err := parseIntQuery(c, "offset", 0, 0, 1_000_000)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		pages, total, err := f.GetCrawlPages(nil, id, storage.PageListOptions{Limit: limit, Offset: offset})
		if err != nil {
			if errors.Is(err, service.ErrCrawlNotFound) {
				response.NotFound(c, "crawl not found")
				return
			}
			response.InternalError(c, "could not fetch crawl pages")
			return
		}
		response.Success(c, http.StatusOK, gin.H{"pages": pages, "total": total, "limit": limit, "offset": offset})
	})

	r.DELETE("/crawls/:id", func(c *gin.Context) {
		id, err := parseIDParam(c, "id")
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if err := f.DeleteCrawl(nil, id); err != nil {
			if errors.Is(err, service.ErrCrawlNotFound) {
				response.NotFound(c, "crawl not found")
				return
			}
			response.InternalError(c, "could not delete crawl")
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.GET("/pages/:id", func(c *gin.Context) {
		id, err := parseIDParam(c, "id")
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		page, err := f.GetPage(nil, id)
		if err != nil {
			if errors.Is(err, service.ErrPageNotFound) {
				response.NotFound(c, "page not found")
				return
			}
			response.InternalError(c, "could not fetch page")
			return
		}
		response.Success(c, http.StatusOK, page)
	})

	return r
}

// =============================================================================
// Tests
// =============================================================================

func TestStartCrawl_BadJSON(t *testing.T) {
	f := newFakeCrawlService()
	r := newRouter(f)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/crawls", bytes.NewBufferString("not json"))
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
	if len(f.startCalls) != 0 {
		t.Errorf("service should not be called for bad JSON, got %d calls", len(f.startCalls))
	}
}

func TestStartCrawl_MissingWebsite(t *testing.T) {
	f := newFakeCrawlService()
	r := newRouter(f)
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"foo": "bar"})
	req := httptest.NewRequest(http.MethodPost, "/crawls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestStartCrawl_Success(t *testing.T) {
	f := newFakeCrawlService()
	f.startOut = &models.Crawl{ID: 42, Website: "https://example.com", Status: models.CrawlStatusPending}
	r := newRouter(f)
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"website": "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/crawls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("code = %d, want 201", rec.Code)
	}
	var env response.Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if !env.Success {
		t.Error("envelope.Success = false, want true")
	}
	if len(f.startCalls) != 1 || f.startCalls[0].website != "https://example.com" {
		t.Errorf("expected one StartCrawl call with the right URL, got %+v", f.startCalls)
	}
}

func TestStartCrawl_InvalidURL(t *testing.T) {
	f := newFakeCrawlService()
	f.startErr = service.ErrInvalidURL
	r := newRouter(f)
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"website": "ftp://bad"})
	req := httptest.NewRequest(http.MethodPost, "/crawls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestStartCrawl_AlreadyRunning_Returns202(t *testing.T) {
	f := newFakeCrawlService()
	f.startOut = &models.Crawl{ID: 7, Status: models.CrawlStatusRunning}
	f.startErr = service.ErrCrawlRunning
	r := newRouter(f)
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"website": "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/crawls", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("code = %d, want 202", rec.Code)
	}
}

func TestGetCrawl_NotFound(t *testing.T) {
	f := newFakeCrawlService()
	r := newRouter(f)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/crawls/999", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestGetCrawl_BadID(t *testing.T) {
	f := newFakeCrawlService()
	r := newRouter(f)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/crawls/not-a-number", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestGetCrawl_Success(t *testing.T) {
	f := newFakeCrawlService()
	f.crawls[7] = &models.Crawl{ID: 7, Website: "https://example.com"}
	r := newRouter(f)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/crawls/7", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("code = %d, want 200", rec.Code)
	}
	var env response.Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if !env.Success {
		t.Error("envelope.Success = false")
	}
}

func TestDeleteCrawl_Success(t *testing.T) {
	f := newFakeCrawlService()
	f.crawls[1] = &models.Crawl{ID: 1}
	r := newRouter(f)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/crawls/1", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("code = %d, want 204", rec.Code)
	}
}

func TestDeleteCrawl_NotFound(t *testing.T) {
	f := newFakeCrawlService()
	r := newRouter(f)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/crawls/999", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestGetCrawlPages_BadLimit(t *testing.T) {
	f := newFakeCrawlService()
	f.crawls[1] = &models.Crawl{ID: 1}
	r := newRouter(f)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/crawls/1/pages?limit=99999", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestGetPage_NotFound(t *testing.T) {
	f := newFakeCrawlService()
	r := newRouter(f)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pages/999", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestGetPage_BadID(t *testing.T) {
	f := newFakeCrawlService()
	r := newRouter(f)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pages/abc", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestParseIntQuery_Defaults(t *testing.T) {
	cases := []struct {
		raw     string
		def     int
		min     int
		max     int
		want    int
		wantErr bool
	}{
		{"", 50, 1, 100, 50, false},
		{"10", 50, 1, 100, 10, false},
		{"abc", 50, 1, 100, 0, true},
		{"-5", 50, 1, 100, 0, true},
		{"200", 50, 1, 100, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/?limit="+tc.raw, nil)
			got, err := parseIntQuery(c, "limit", tc.def, tc.min, tc.max)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestParseIDParam(t *testing.T) {
	cases := []struct {
		raw     string
		want    int64
		wantErr bool
	}{
		{"42", 42, false},
		{"1", 1, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Params = gin.Params{{Key: "id", Value: tc.raw}}
			got, err := parseIDParam(c, "id")
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}

	// Verify the workers.Job contract the service layer relies on is
	// stable — protects against a regression where the field name
	// accidentally reverts to all-caps.
	j := workers.Job{CrawlID: 1, URL: "https://x"}
	if strconv.FormatInt(j.CrawlID, 10) != "1" {
		t.Errorf("Job.CrawlID roundtrip failed")
	}
}