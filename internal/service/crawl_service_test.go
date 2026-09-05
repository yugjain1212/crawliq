package service

import "testing"

func TestNormalizeWebsite(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "already normalized https",
			input: "https://example.com",
			want:  "https://example.com",
		},
		{
			name:  "trailing slash stripped",
			input: "https://example.com/",
			want:  "https://example.com",
		},
		{
			name:  "path stripped",
			input: "https://example.com/some/path?q=1",
			want:  "https://example.com",
		},
		{
			name:  "uppercase host normalized via url.Parse — case preserved",
			input: "https://EXAMPLE.com",
			want:  "https://EXAMPLE.com", // url.Parse preserves case
		},
		{
			name:  "missing scheme defaults to https",
			input: "example.com",
			want:  "https://example.com",
		},
		{
			name:  "whitespace trimmed",
			input: "  https://example.com  ",
			want:  "https://example.com",
		},
		{
			name:  "http preserved when given explicitly",
			input: "http://example.com",
			want:  "http://example.com",
		},
		{
			name:    "empty string rejected",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace-only rejected",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "unsupported scheme rejected",
			input:   "ftp://example.com",
			wantErr: true,
		},
		{
			name:    "missing host rejected",
			input:   "https://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeWebsite(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeWebsite(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("normalizeWebsite(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}