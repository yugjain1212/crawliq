package sitemap

import "encoding/xml"

type SitemapType string

const (
	SitemapTypeIndex   SitemapType = "index"
	SitemapTypeURLSet  SitemapType = "urlset"
	SitemapTypeUnknown SitemapType = "unknown"
)

type URLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []SitemapURL `xml:"url"`
}

type SitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type SitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	Sitemaps []SitemapEntry `xml:"sitemap"`
}

type SitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type DiscoveryResult struct {
	URLs          []string
	SitemapsFound int
	Errors        []string
}
