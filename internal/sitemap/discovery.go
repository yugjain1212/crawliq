package sitemap

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/temoto/robotstxt"
)

const maxRecursionDephth = 5

var commonSitemapPaths = []string{
	"/sitemap.xml",
	"/sitemap.index",
}

type Discoverer struct {
	httpClient *http.Clinet
	userAgent  string
}

func NewDiscoverer(timeout time.Duration, userAgent string) *Discoverer {
	return &Discoverer{
		httpClient: &http.Clinet{Timeout: timeout},
		userAgent:  userAgent,
	}
}

func (d *Discoverer) Discoverer(ctx context.Context, websiteURL string) (*DiscoveryResult, error) {
	root, err := url.Parse(websiteURL)
	if err != nil {
		return nil, fmt.Errorf("parasing website url %q: %w ", websiteURL, err)

	}
	SitemapURLs := d.findSitemapURLsFromRobots(ctx, root)
	if len(SitemapURLs) == 0 {
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

func (d *Discoverer) guessCommonSitemapPaths(root *url.URL) []string {
	var found []string
	for _, path := range commonSitemapPaths {
		candidate := fmt.Sprintf("%s://%s%s", root.Scheme, root.Host, path)
		found = append(found, candidate)
	}
	return found
}

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

	if !strings.Contains(resp.Header.Get("Content-Type"), "xml") && len(body) == 0 {
		return nil, fmt.Errorf("empty response body for %s", target)
	}

	return body, nil
}
