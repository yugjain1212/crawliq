package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	// Tests share the default Gin mode setting; force test mode so we
	// don't get noisy stderr output for each router bring-up.
	gin.SetMode(gin.TestMode)
}

func TestSuccess_EnvelopeShape(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	Success(c, http.StatusOK, map[string]string{"hello": "world"})

	if rec.Code != http.StatusOK {
		t.Errorf("Success() code = %d, want %d", rec.Code, http.StatusOK)
	}

	var body Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if !body.Success {
		t.Error("Success() body.Success = false, want true")
	}
	if body.Error != nil {
		t.Errorf("Success() body.Error = %+v, want nil", body.Error)
	}
	if body.Data == nil {
		t.Error("Success() body.Data = nil, want non-nil")
	}
}

func TestError_EnvelopeShape(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	Error(c, http.StatusNotFound, CodeNotFound, "thing missing")

	if rec.Code != http.StatusNotFound {
		t.Errorf("Error() code = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var body Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	if body.Success {
		t.Error("Error() body.Success = true, want false")
	}
	if body.Data != nil {
		t.Errorf("Error() body.Data = %+v, want nil", body.Data)
	}
	if body.Error == nil {
		t.Fatal("Error() body.Error = nil, want non-nil")
	}
	if body.Error.Code != CodeNotFound {
		t.Errorf("Error() code = %q, want %q", body.Error.Code, CodeNotFound)
	}
	if body.Error.Message != "thing missing" {
		t.Errorf("Error() message = %q, want %q", body.Error.Message, "thing missing")
	}
}

func TestNotFound_Helper(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	NotFound(c, "crawl 7")

	if rec.Code != http.StatusNotFound {
		t.Errorf("NotFound() code = %d, want 404", rec.Code)
	}

	var body Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error == nil || body.Error.Code != CodeNotFound {
		t.Errorf("NotFound() did not produce CodeNotFound error: %+v", body.Error)
	}
}

func TestBadRequest_Helper(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	BadRequest(c, "nope")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("BadRequest() code = %d, want 400", rec.Code)
	}

	var body Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error == nil || body.Error.Code != CodeBadRequest {
		t.Errorf("BadRequest() did not produce CodeBadRequest error: %+v", body.Error)
	}
}

func TestInternalError_GenericMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	InternalError(c, "")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("InternalError() code = %d, want 500", rec.Code)
	}

	var body Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Error == nil {
		t.Fatal("InternalError() error = nil, want non-nil")
	}
	if body.Error.Code != CodeInternal {
		t.Errorf("InternalError() code = %q, want %q", body.Error.Code, CodeInternal)
	}
	// Empty input → caller-supplied generic message used.
	if body.Error.Message == "" {
		t.Error("InternalError() with empty message should still have a non-empty body message")
	}
}

func TestStableCodes(t *testing.T) {
	// Guard against accidental code-name changes — these are part of
	// the API contract (clients branch on them).
	want := map[string]string{
		"CodeValidation": CodeValidation,
		"CodeNotFound":   CodeNotFound,
		"CodeInternal":   CodeInternal,
		"CodeConflict":   CodeConflict,
		"CodeBadRequest": CodeBadRequest,
	}
	expected := map[string]string{
		"CodeValidation": "VALIDATION_ERROR",
		"CodeNotFound":   "NOT_FOUND",
		"CodeInternal":   "INTERNAL_ERROR",
		"CodeConflict":   "CONFLICT",
		"CodeBadRequest": "BAD_REQUEST",
	}
	for name, code := range want {
		if code != expected[name] {
			t.Errorf("%s = %q, want %q", name, code, expected[name])
		}
	}
}