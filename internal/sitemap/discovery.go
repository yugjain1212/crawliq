package sitemap

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/temoto/robotstxt"
)

// maxRecursionDepth is a hard ceiling on nested sitemap index expansion
// so a (malicious or buggy) sitemap that points to itself can't loop
// the crawler forever.
const maxRecursionDepth = 5

// commonSitemapPaths is the fallback list of locations we try when
// robots.txt didn't declare a sitemap (or robots.txt itself couldn't be
// fetched). Sitemap.xml is the de-facto standard; sitemap_index.xml is
// a common alternative used by some CMSs.
var commonSitemapPaths = []string{
	"/sitemap.xml",
	"/sitemap_index.xml",
}

// Discoverer is the entry point for "given a website URL, give me every
// page URL the site wants crawled". It looks at robots.txt first, then
// falls back to common paths, then recursively resolves any sitemap
// indexes it finds along the way.
type Discoverer struct {
	httpClient *http.Client
	userAgent  string
}

func NewDiscoverer(timeout time.Duration, userAgent string) *Discoverer {
	return &Discoverer{
		httpClient: &http.Client{Timeout: timeout},
		userAgent:  userAgent,
	}
}

// Discover runs the full discovery pipeline and returns every URL it
// could find across every reachable sitemap (de-duplicated). The Errors
// slice carries non-fatal failures (a single unreachable sitemap, etc.)
// so callers can surface them in the API response or logs.
func (d *Discoverer) Discover(ctx context.Context, websiteURL string) (*DiscoveryResult, error) {
	root, err := url.Parse(websiteURL)
	if err != nil {
		return nil, fmt.Errorf("parsing website url %q: %w", websiteURL, err)
	}

	sitemapURLs := d.findSitemapURLsFromRobots(ctx, root)
	if len(sitemapURLs) == 0 {
		sitemapURLs = d.guessCommonSitemapPaths(root)
	}

	result := &DiscoveryResult{}
	seen := make(map[string]bool)

	for _, sitemapURL := range sitemapURLs {
		d.resolveSitemap(ctx, sitemapURL, 0, result, seen)
	}

	log.Info().
		Str("website", websiteURL).
		Int("urls_found", len(result.URLs)).
		Int("sitemaps_found", result.SitemapsFound).
		Int("errors", len(result.Errors)).
		Msg("sitemap discovery complete")

	return result, nil
}

// findSitemapURLsFromRobots fetches /robots.txt and extracts any
// "Sitemap:" directives. Returns nil (no error) if robots.txt is
// missing or unparseable — that's a normal condition, not a failure,
// and the caller will fall back to common sitemap paths.
func (d *Discoverer) findSitemapURLsFromRobots(ctx context.Context, root *url.URL) []string {
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", root.Scheme, root.Host)

	body, err := d.fetch(ctx, robotsURL)
	if err != nil {
		log.Debug().Err(err).Str("url", robotsURL).Msg("robots.txt not fetched, will fall back to common paths")
		return nil
	}

	parsed, err := robotstxt.FromBytes(body)
	if err != nil {
		log.Debug().Err(err).Str("url", robotsURL).Msg("robots.txt could not be parsed")
		return nil
	}

	return parsed.Sitemaps
}

// guessCommonSitemapPaths produces the candidate URLs we try when
// robots.txt didn't tell us where the sitemap lives.
func (d *Discoverer) guessCommonSitemapPaths(root *url.URL) []string {
	found := make([]string, 0, len(commonSitemapPaths))
	for _, path := range commonSitemapPaths {
		candidate := fmt.Sprintf("%s://%s%s", root.Scheme, root.Host, path)
		found = append(found, candidate)
	}
	return found
}

// resolveSitemap fetches a single sitemap URL, classifies it
// (urlset vs. sitemap index), and either extracts page URLs or recurses
// into the children. Per-sitemap errors are recorded into the result
// rather than aborting the whole discovery — one broken sitemap
// shouldn't lose us the URLs from the others.
func (d *Discoverer) resolveSitemap(ctx context.Context, sitemapURL string, depth int, result *DiscoveryResult, seen map[string]bool) {
	if depth > maxRecursionDepth {
		result.Errors = append(result.Errors, fmt.Sprintf("max recursion depth exceeded at %s", sitemapURL))
		return
	}

	body, err := d.fetch(ctx, sitemapURL)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("fetching %s: %v", sitemapURL, err))
		return
	}

	sitemapType, urlSet, index, err := Parse(body)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("parsing %s: %v", sitemapURL, err))
		return
	}

	result.SitemapsFound++

	switch sitemapType {
	case SitemapTypeURLSet:
		for _, loc := range urlSet.ExtractLocs() {
			if !seen[loc] {
				seen[loc] = true
				result.URLs = append(result.URLs, loc)
			}
		}

	case SitemapTypeIndex:
		for _, childURL := range index.ExtractLocs() {
			d.resolveSitemap(ctx, childURL, depth+1, result, seen)
		}
	}
}

// fetch is the discoverer's tiny HTTP helper. Anything outside 2xx is
// an error; empty bodies are also rejected because an empty "sitemap"
// is almost certainly a 404 page that didn't return a 404 status.
func (d *Discoverer) fetch(ctx context.Context, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("User-Agent", d.userAgent)

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, target)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body for %s", target)
	}

	// We intentionally don't require an XML content-type because some
	// servers send text/plain or application/octet-stream for sitemaps;
	// presence of any body is enough for the parser to classify.

	return body, nil
}