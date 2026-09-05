package sitemap

import (
	"testing"
)

func TestParse_URLSet(t *testing.T) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/</loc></url>
  <url><loc>https://example.com/about</loc></url>
  <url><loc>https://example.com/contact</loc></url>
</urlset>`)

	sitemapType, urlSet, idx, err := Parse(body)
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}
	if sitemapType != SitemapTypeURLSet {
		t.Errorf("Parse() type = %v, want %v", sitemapType, SitemapTypeURLSet)
	}
	if idx != nil {
		t.Error("Parse() returned non-nil index for urlset")
	}
	if urlSet == nil {
		t.Fatal("Parse() returned nil urlset")
	}

	locs := urlSet.ExtractLocs()
	want := []string{"https://example.com/", "https://example.com/about", "https://example.com/contact"}
	if len(locs) != len(want) {
		t.Fatalf("ExtractLocs() length = %d, want %d", len(locs), len(want))
	}
	for i, w := range want {
		if locs[i] != w {
			t.Errorf("ExtractLocs()[%d] = %q, want %q", i, locs[i], w)
		}
	}
}

func TestParse_SitemapIndex(t *testing.T) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>https://example.com/sitemap-pages.xml</loc></sitemap>
  <sitemap><loc>https://example.com/sitemap-posts.xml</loc></sitemap>
</sitemapindex>`)

	sitemapType, urlSet, idx, err := Parse(body)
	if err != nil {
		t.Fatalf("Parse() unexpected error: %v", err)
	}
	if sitemapType != SitemapTypeIndex {
		t.Errorf("Parse() type = %v, want %v", sitemapType, SitemapTypeIndex)
	}
	if urlSet != nil {
		t.Error("Parse() returned non-nil urlset for sitemapindex")
	}
	if idx == nil {
		t.Fatal("Parse() returned nil index")
	}

	locs := idx.ExtractLocs()
	if len(locs) != 2 {
		t.Fatalf("ExtractLocs() length = %d, want 2", len(locs))
	}
	if locs[0] != "https://example.com/sitemap-pages.xml" {
		t.Errorf("ExtractLocs()[0] = %q", locs[0])
	}
}

func TestParse_UnknownRoot(t *testing.T) {
	body := []byte(`<?xml version="1.0"?><html><body>not a sitemap</body></html>`)

	_, _, _, err := Parse(body)
	if err == nil {
		t.Error("Parse() expected error for unknown root element, got nil")
	}
}

func TestParse_MalformedXML(t *testing.T) {
	body := []byte(`<urlset><url><loc>broken`)

	_, _, _, err := Parse(body)
	if err == nil {
		t.Error("Parse() expected error for malformed XML, got nil")
	}
}

func TestURLSet_ExtractLocs_SkipsEmpty(t *testing.T) {
	u := &URLSet{
		URLs: []SitemapURL{
			{Loc: "https://a.example/"},
			{Loc: ""},
			{Loc: "https://b.example/"},
		},
	}
	got := u.ExtractLocs()
	if len(got) != 2 {
		t.Errorf("ExtractLocs() length = %d, want 2 (empty loc should be skipped)", len(got))
	}
}