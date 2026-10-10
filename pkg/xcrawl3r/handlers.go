package xcrawl3r

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/gocolly/colly/v2"
)

func (c *Crawler) registerHandlers(collector *colly.Collector, scope *scope, results chan<- Result) {
	collector.OnRequest(func(request *colly.Request) {
		if match := c.fileURLsNotToRequestExtRegex.MatchString(path.Ext(request.URL.Path)); match {
			request.Abort()
		}
	})

	collector.OnError(func(response *colly.Response, err error) {
		results <- Result{
			Type:  ResultError,
			Error: fmt.Errorf("error requesting %s: %w", response.Request.URL.String(), err),
		}
	})

	collector.OnResponse(func(response *colly.Response) {
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

		replacer := strings.NewReplacer(
			"*", "",
			`\u002f`, "/",
			`\u0026`, "&",
		)

		body := replacer.Replace(string(response.Body))

		for _, link := range c.urlExtractorRegex.FindAllString(body, -1) {
			// On HTML pages only absolute links are extracted (e.g. from inline
			// scripts); relative ones are covered by the OnHTML attribute handlers
			// and would otherwise match tag fragments like "</script>".
			if isHTML && !strings.Contains(link, "://") {
				continue
			}

			c.processLink(response.Request, link, scope, results)
		}
	})

	collector.OnHTML("[href]", func(e *colly.HTMLElement) {
		c.processLink(e.Request, e.Attr("href"), scope, results)
	})

	collector.OnHTML("[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")

		if URL := e.Request.AbsoluteURL(link); scope.allows(URL) && strings.Contains(URL, ".min.") {
			companion := strings.ReplaceAll(URL, ".min.", ".")

			c.visit(companion, results, func() error {
				return e.Request.Visit(companion)
			})
		}

		c.processLink(e.Request, link, scope, results)
	})
}

func (c *Crawler) processLink(request *colly.Request, link string, scope *scope, results chan<- Result) {
	URL := request.AbsoluteURL(link)

	if !scope.allows(URL) || c.alreadySeen(URL) {
		return
	}

	results <- Result{
		Type:  ResultURL,
		Value: URL,
	}

	c.visit(URL, results, func() error {
		return request.Visit(URL)
	})
}

func (c *Crawler) visit(URL string, results chan<- Result, do func() error) {
	if err := do(); err != nil && !isExpectedVisitError(err) {
		results <- Result{
			Type:  ResultError,
			Error: fmt.Errorf("error visiting %s: %w", URL, err),
		}
	}
}

func (c *Crawler) alreadySeen(URL string) (seen bool) {
	key := URL

	if index := strings.IndexByte(key, '#'); index >= 0 {
		key = key[:index]
	}

	_, seen = c.seen.LoadOrStore(key, struct{}{})

	return
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
