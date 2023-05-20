package xcrawl3r

import (
	"fmt"

	hqurl "github.com/hueristiq/hqgoutils/url"
	sitemap "github.com/oxffaa/gopher-parse-sitemap"
)

func (crawler *Crawler) sitemapParsing(parsedURL *hqurl.URL) (URLsChannel chan URL) {
	URLsChannel = make(chan URL)

	go func() {
		defer close(URLsChannel)

		sitemapPaths := []string{
			"/sitemap.xml",
			"/sitemap_news.xml",
			"/sitemap_index.xml",
			"/sitemap-index.xml",
			"/sitemapindex.xml",
			"/sitemap-news.xml",
			"/post-sitemap.xml",
			"/page-sitemap.xml",
			"/portfolio-sitemap.xml",
			"/home_slider-sitemap.xml",
			"/category-sitemap.xml",
			"/author-sitemap.xml",
		}

		for _, path := range sitemapPaths {
			sitemapURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, path)

			err := sitemap.ParseFromSite(sitemapURL, func(entry sitemap.Entry) (err error) {
				smURL := entry.GetLocation()

				URLsChannel <- URL{Source: "sitemap", Value: smURL}

				if err = crawler.PageCollector.Visit(smURL); err != nil {
					return
				}

				return
			})
			if err != nil {
				continue
			}

			URLsChannel <- URL{Source: "known", Value: sitemapURL}
		}
	}()

	return
}
