package crawler

import (
	"strings"
	"testing"
)

func TestParseHTML_Title(t *testing.T) {
	html := []byte(`<!DOCTYPE html>
<html>
<head><title>  Hello World  </title></head>
<body><h1>Hi</h1></body>
</html>`)

	parsed, err := ParseHTML(html)
	if err != nil {
		t.Fatalf("ParseHTML() unexpected error: %v", err)
	}
	if parsed.Title != "Hello World" {
		t.Errorf("ParseHTML() title = %q, want %q (whitespace should be trimmed)", parsed.Title, "Hello World")
	}
}

func TestParseHTML_NoTitle(t *testing.T) {
	html := []byte(`<!DOCTYPE html><html><body><p>no title here</p></body></html>`)

	parsed, err := ParseHTML(html)
	if err != nil {
		t.Fatalf("ParseHTML() unexpected error: %v", err)
	}
	if parsed.Title != "" {
		t.Errorf("ParseHTML() title = %q, want empty string", parsed.Title)
	}
}

func TestParseHTML_OnlyFirstTitle(t *testing.T) {
	html := []byte(`<html><head>
		<title>First Title</title>
	</head><body>
		<title>Body Title — should be ignored</title>
	</body></html>`)

	parsed, err := ParseHTML(html)
	if err != nil {
		t.Fatalf("ParseHTML() unexpected error: %v", err)
	}
	if parsed.Title != "First Title" {
		t.Errorf("ParseHTML() title = %q, want %q (only first <title> should win)", parsed.Title, "First Title")
	}
}

func TestIsHTML(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"TEXT/HTML", true},
		{"application/xhtml+xml", true},
		{"application/json", false},
		{"text/plain", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			if got := IsHTML(tt.contentType); got != tt.want {
				t.Errorf("IsHTML(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestIsHTML_PartialMatch(t *testing.T) {
	// Some servers send unusual combinations; we only check that
	// "text/html" is a substring.
	if !IsHTML("application/something+text/html; charset=utf-8") {
		t.Error("IsHTML should match when 'text/html' appears as substring")
	}
}

// Sanity check: ParseHTML should produce a non-nil result even for
// empty input (goquery is permissive).
func TestParseHTML_EmptyBody(t *testing.T) {
	parsed, err := ParseHTML(nil)
	if err != nil {
		t.Fatalf("ParseHTML(nil) unexpected error: %v", err)
	}
	if parsed == nil {
		t.Fatal("ParseHTML(nil) returned nil result")
	}
	if strings.TrimSpace(parsed.Title) != "" {
		t.Errorf("ParseHTML(nil) title = %q, want empty", parsed.Title)
	}
}