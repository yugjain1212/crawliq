package models

import "testing"

func intPtr(v int) *int    { return &v }
func strPtr(v string) *string { return &v }

func TestPage_Succeeded(t *testing.T) {
	tests := []struct {
		name string
		page Page
		want bool
	}{
		{
			name: "200 OK succeeds",
			page: Page{StatusCode: intPtr(200)},
			want: true,
		},
		{
			name: "299 (last 2xx) succeeds",
			page: Page{StatusCode: intPtr(299)},
			want: true,
		},
		{
			name: "301 redirect succeeds (it's a fetch success even if not final)",
			page: Page{StatusCode: intPtr(301)},
			want: true,
		},
		{
			name: "399 (last 3xx) succeeds",
			page: Page{StatusCode: intPtr(399)},
			want: true,
		},
		{
			name: "404 fails",
			page: Page{StatusCode: intPtr(404)},
			want: false,
		},
		{
			name: "500 fails",
			page: Page{StatusCode: intPtr(500)},
			want: false,
		},
		{
			name: "nil status code (fetch itself failed) fails",
			page: Page{StatusCode: nil},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.page.Succeeded(); got != tt.want {
				t.Errorf("Succeeded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPage_SucceededWithTitle(t *testing.T) {
	// A common sanity check: a 200 page with a parsed title should
	// still report Succeeded() == true.
	p := Page{
		StatusCode: intPtr(200),
		Title:      strPtr("Welcome to Example"),
	}
	if !p.Succeeded() {
		t.Error("expected Succeeded() == true for 200 page with title")
	}
}