package xcrawl3r

import (
	"compress/gzip"
	"context"
	"errors"
	"strings"

	"github.com/gocolly/colly/v2"
)

// sitemapPaths lists the fixed sitemap probes tried on every target host.
var sitemapPaths = []string{
	"/sitemap.xml",
	"/sitemap.xml.gz",
	"/sitemap_news.xml",
	"/sitemap_index.xml",
	"/sitemap-index.xml",
	"/sitemapindex.xml",
	"/sitemap-news.xml",
	"/wp-sitemap.xml",
}

type robotsRules struct {
	paths    []string
	sitemaps []string
}

func parseRobots(body []byte) (rules robotsRules) {
	content := strings.TrimPrefix(string(body), "\xef\xbb\xbf")

	for line := range strings.Lines(content) {
		if index := strings.IndexByte(line, '#'); index >= 0 {
			line = line[:index]
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		value = strings.TrimSpace(value)

		switch strings.ToLower(strings.TrimSpace(key)) {
		case "disallow", "allow":
			if value == "" || value == "/" {
				continue
			}

			if index := strings.IndexAny(value, "*$"); index >= 0 {
				value = value[:index]
			}

			if value == "" {
				continue
			}

			rules.paths = append(rules.paths, value)
		case "sitemap":
			if value == "" {
				continue
			}

			rules.sitemaps = append(rules.sitemaps, value)
		}
	}

	return
}

func isSitemapPath(rawPath string) (sitemap bool) {
	rawPath = strings.ToLower(rawPath)

	if !strings.Contains(rawPath, "sitemap") {
		return
	}

	return strings.HasSuffix(rawPath, ".xml") || strings.HasSuffix(rawPath, ".xml.gz")
}

// isNonGzippedXMLGZProbe reports whether err is the expected gzip failure from
// probing a .xml.gz path that does not actually serve gzip.
func isNonGzippedXMLGZProbe(rawURL string, err error) (expected bool) {
	if !strings.HasSuffix(strings.ToLower(rawURL), ".xml.gz") {
		return
	}

	return errors.Is(err, gzip.ErrHeader)
}

func (c *Crawler) handleRobots(ctx context.Context, response *colly.Response, scope *scope, results chan<- Result) {
	rules := parseRobots(response.Body)

	for _, sitemap := range rules.sitemaps {
		if ctx.Err() != nil {
			return
		}

		if !scope.allows(sitemap) {
			continue
		}

		c.visit(ctx, sitemap, results, func() error {
			return response.Request.Visit(sitemap)
		})
	}

	for _, path := range rules.paths {
		c.processLink(ctx, response.Request, path, "robots", scope, results)
	}
}
