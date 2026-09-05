package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the uniform JSON shape every API response uses. Having a
// single envelope (instead of returning raw data for successes and a
// different shape for errors) keeps the client-side code predictable:
// every response has the same top-level keys.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo carries the machine-readable error code separately from the
// human-readable message, so clients can branch on the code without
// parsing the message string.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Success writes a 2xx response with the given payload wrapped in the
// standard envelope.
func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, Envelope{
		Success: true,
		Data:    data,
	})
}

// Error writes a non-2xx response with a structured error body. Use
// StatusInternalServerError sparingly — most callers should reach for
// the typed helpers below so the response code stays in one place.
func Error(c *gin.Context, status int, code string, message string) {
	c.JSON(status, Envelope{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// Stable error codes the client can branch on without parsing messages.
// Kept in one place so a typo doesn't silently break client logic.
const (
	CodeValidation = "VALIDATION_ERROR"
	CodeNotFound   = "NOT_FOUND"
	CodeInternal   = "INTERNAL_ERROR"
	CodeConflict   = "CONFLICT"
	CodeBadRequest = "BAD_REQUEST"
)

// BadRequest sends a 400 with a stable code and the supplied message.
// Use this for malformed input — bad URL syntax, missing fields, etc.
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, CodeBadRequest, message)
}

// NotFound sends a 404. Typically used when a repository returns one of
// the sentinel "not found" errors.
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, CodeNotFound, message)
}

// InternalError sends a 500 with a generic user-facing message. The real
// error should already have been logged at the layer where it
// originated; we don't echo internal error text to the client.
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "An internal error occurred. Please try again later."
	}
	Error(c, http.StatusInternalServerError, CodeInternal, message)
}