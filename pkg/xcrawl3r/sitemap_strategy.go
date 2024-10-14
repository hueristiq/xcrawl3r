package xcrawl3r

import (
	"fmt"
	"net/http"

	hqgohttp "github.com/hueristiq/hq-go-http"
	hqgourl "github.com/hueristiq/hq-go-url"
	sitemap "github.com/hueristiq/xcrawl3r/pkg/parser/sitemap"
)

func (crawler *Crawler) sitemapParsing(parsedURL *hqgourl.URL) <-chan Result {
	results := make(chan Result)

	go func() {
		defer close(results)

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

			if err := crawler.parseSitemap(sitemapURL, results); err != nil {
				result := Result{
					Type:   ResultError,
					Source: "known:sitemap",
					Error:  err,
				}

				results <- result

				continue
			}

			result := Result{
				Type:   ResultURL,
				Source: "known:sitemap",
				Value:  sitemapURL,
			}

			results <- result
		}
	}()

	return results
}

func (crawler *Crawler) parseSitemap(URL string, results chan Result) (err error) {
	var res *http.Response

	res, err = hqgohttp.Get(URL)
	if err != nil {
		return
	}

	if err = sitemap.Parse(res.Body, func(entry sitemap.Entry) (err error) {
		sitemapEntryURL := entry.GetLocation()

		result := Result{
			Type:   ResultURL,
			Source: "sitemap",
			Value:  sitemapEntryURL,
		}

		results <- result

		if entry.GetType() == sitemap.EntryTypeSitemap {
			return crawler.parseSitemap(sitemapEntryURL, results)
		}

		if err = crawler.PageCollector.Visit(sitemapEntryURL); err != nil {
			result := Result{
				Type:   ResultError,
				Source: "sitemap",
				Error:  err,
			}

			results <- result

			err = nil

			return
		}

		return
	}); err != nil {
		return
	}

	res.Body.Close()

	return
}
