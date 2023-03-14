package hqcrawl3r

import (
	"fmt"

	sitemap "github.com/oxffaa/gopher-parse-sitemap"
)

func (crawler *Crawler) ParseSitemap() {
	sitemapPaths := []string{"/sitemap.xml", "/sitemap_news.xml", "/sitemap_index.xml", "/sitemap-index.xml", "/sitemapindex.xml", "/sitemap-news.xml", "/post-sitemap.xml", "/page-sitemap.xml", "/portfolio-sitemap.xml", "/home_slider-sitemap.xml", "/category-sitemap.xml", "/author-sitemap.xml"}

	for _, path := range sitemapPaths {
		sitemapURL := fmt.Sprintf("%s://%s%s", crawler.URL.Scheme, crawler.URL.Host, path)

		if _, exists := visitedURLs.Load(sitemapURL); exists {
			continue
		}

		_ = sitemap.ParseFromSite(sitemapURL, func(entry sitemap.Entry) error {
			crawler.PageCollector.Visit(entry.GetLocation())
			return nil
		})

		visitedURLs.Store(sitemapURL, struct{}{})
	}
}
