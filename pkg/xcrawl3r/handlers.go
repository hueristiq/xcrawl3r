package xcrawl3r

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/gocolly/colly/v2"
)

func (c *Crawler) registerHandlers(ctx context.Context, collector *colly.Collector, scope *scope, results chan<- Result) {
	collector.OnRequest(func(request *colly.Request) {
		if ctx.Err() != nil {
			request.Abort()

			return
		}

		if match := c.fileURLsNotToRequestExtRegex.MatchString(path.Ext(request.URL.Path)); match {
			request.Abort()
		}
	})

	collector.OnError(func(response *colly.Response, err error) {
		if ctx.Err() != nil {
			return
		}

		emit(ctx, results, Result{
			Type:  ResultError,
			Error: fmt.Errorf("error requesting %s: %w", response.Request.URL.String(), err),
		})
	})

	collector.OnResponse(func(response *colly.Response) {
		if strings.EqualFold(response.Request.URL.Path, "/robots.txt") {
			c.handleRobots(ctx, response, scope, results)

			return
		}

		contentType := response.Headers.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(response.Body)
		}

		mediatype, _, _ := strings.Cut(contentType, ";")
		mediatype = strings.TrimSpace(strings.ToLower(mediatype))

		if !isExtractableMediaType(mediatype) && !c.fileURLsToRequestExtRegex.MatchString(path.Ext(response.Request.URL.Path)) {
			return
		}

		isHTML := mediatype == "text/html" || mediatype == "application/xhtml+xml"

		via := "regex"

		if isSitemapPath(response.Request.URL.Path) {
			via = "sitemap"
		}

		replacer := strings.NewReplacer(
			"*", "",
			`\u002f`, "/",
			`\u0026`, "&",
			`\u003c`, "<",
			`\u003C`, "<",
			`\u003e`, ">",
			`\u003E`, ">",
		)

		body := replacer.Replace(string(response.Body))

		for _, loc := range c.urlExtractorRegex.FindAllStringIndex(body, -1) {
			link := body[loc[0]:loc[1]]

			// A relative match starting right after '<' is the tail of an
			// HTML/XML tag ("/div" in "</div>", "/loc" in "</loc>"), not a
			// URL.
			if loc[0] > 0 && body[loc[0]-1] == '<' && !strings.Contains(link, "://") {
				continue
			}

			// On HTML pages only absolute links are extracted (e.g. from inline
			// scripts); relative ones are covered by the OnHTML attribute handlers
			// and would otherwise match tag fragments like "</script>".
			if isHTML && !strings.Contains(link, "://") {
				continue
			}

			c.processLink(ctx, response.Request, link, via, scope, results)
		}

		if isHTML && c.headless != nil {
			var renderedURLs []string

			renderedURLs, err := c.render(ctx, response.Request.URL.String())
			if err != nil {
				return
			}

			for _, link := range renderedURLs {
				c.processLink(ctx, response.Request, link, "headless", scope, results)
			}
		}
	})

	collector.OnHTML("[href]", func(e *colly.HTMLElement) {
		c.processLink(ctx, e.Request, e.Attr("href"), "href", scope, results)
	})

	collector.OnHTML("[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")

		if URL := e.Request.AbsoluteURL(link); scope.allows(URL) && strings.Contains(URL, ".min.") {
			companion := strings.ReplaceAll(URL, ".min.", ".")

			c.visit(ctx, companion, results, scope, func() error {
				return e.Request.Visit(companion)
			})
		}

		c.processLink(ctx, e.Request, link, "src", scope, results)
	})
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

		c.visit(ctx, sitemap, results, scope, func() error {
			return response.Request.Visit(sitemap)
		})
	}

	for _, path := range rules.paths {
		c.processLink(ctx, response.Request, path, "robots", scope, results)
	}
}

func (c *Crawler) processLink(ctx context.Context, request *colly.Request, link, via string, scope *scope, results chan<- Result) {
	URL := request.AbsoluteURL(link)

	if !scope.allows(URL) || c.alreadySeen(URL) {
		return
	}

	result := Result{
		Type:   ResultURL,
		Value:  URL,
		Source: request.URL.String(),
		Via:    via,
	}

	if !emit(ctx, results, result) {
		return
	}

	c.visit(ctx, URL, results, scope, func() error {
		return request.Visit(URL)
	})
}

func (c *Crawler) visit(ctx context.Context, URL string, results chan<- Result, scope *scope, do func() error) {
	if scope.maxPages > 0 {
		if scope.requestCount.Add(1) > uint32(scope.maxPages) { //nolint:gosec // maxPages validated >= 0 in New(); the > 0 guard here covers the rest.
			return
		}
	}

	err := do()
	if err == nil {
		return
	}

	// Unsuccessful visits (already-visited, depth-exceeded, transport
	// failures) do not consume the page budget.
	scope.requestCount.Add(^uint32(0))

	if !isExpectedVisitError(err) {
		emit(ctx, results, Result{
			Type:  ResultError,
			Error: fmt.Errorf("error visiting %s: %w", URL, err),
		})
	}
}

func (c *Crawler) alreadySeen(URL string) (seen bool) {
	key := normalizeURL(URL, c.opts.ignoreQuery, c.opts.shapeDedup)

	_, seen = c.seen.LoadOrStore(key, struct{}{})

	return
}

func emit(ctx context.Context, results chan<- Result, result Result) (emitted bool) {
	select {
	case results <- result:
		return true
	case <-ctx.Done():
		return false
	}
}

func isExpectedVisitError(err error) (expected bool) {
	var alreadyVisitedError *colly.AlreadyVisitedError

	switch {
	case errors.As(err, &alreadyVisitedError),
		errors.Is(err, colly.ErrMaxDepth),
		errors.Is(err, colly.ErrNoURLFiltersMatch),
		errors.Is(err, colly.ErrForbiddenDomain),
		errors.Is(err, colly.ErrRobotsTxtBlocked):
		return true
	}

	return
}

func isExtractableMediaType(mediatype string) (extractable bool) {
	if strings.HasPrefix(mediatype, "text/") {
		return true
	}

	switch mediatype {
	case "application/json",
		"application/javascript",
		"application/xml",
		"application/xhtml+xml",
		"application/x-www-form-urlencoded":
		return true
	}

	return strings.HasSuffix(mediatype, "+json") || strings.HasSuffix(mediatype, "+xml")
}

func isSitemapPath(rawPath string) (sitemap bool) {
	rawPath = strings.ToLower(rawPath)

	return strings.HasSuffix(rawPath, ".xml") && strings.Contains(rawPath, "sitemap")
}
