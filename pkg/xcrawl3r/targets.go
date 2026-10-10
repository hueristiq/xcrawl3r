package xcrawl3r

import (
	"fmt"
	"strings"

	hqgourlparser "github.com/hueristiq/hq-lib-url-go/parser"
)

var sitemapPaths = []string{
	"/sitemap.xml",
	"/sitemap_news.xml",
	"/sitemap_index.xml",
	"/sitemap-index.xml",
	"/sitemapindex.xml",
	"/sitemap-news.xml",
	"/wp-sitemap.xml",
}

func (c *Crawler) targets(parsedTargetURL *hqgourlparser.URL) (targets []string) {
	targets = []string{parsedTargetURL.String()}

	if strings.Contains(parsedTargetURL.String(), ".min.") {
		targets = append(targets, strings.ReplaceAll(parsedTargetURL.String(), ".min.", "."))
	}

	robotsTXTURL := fmt.Sprintf("%s://%s/robots.txt", parsedTargetURL.Scheme, parsedTargetURL.Host)

	targets = append(targets, robotsTXTURL)

	for _, path := range sitemapPaths {
		sitemapURL := fmt.Sprintf("%s://%s%s", parsedTargetURL.Scheme, parsedTargetURL.Host, path)

		targets = append(targets, sitemapURL)
	}

	return
}
