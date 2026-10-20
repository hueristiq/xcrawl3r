package xcrawl3r

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly/v2"
)

// urlContextRegex matches the text immediately before a relative reference
// inside a CSS url(...) or @import context, e.g. "url(", "url('", `url("`,
// or `@import "`.
var urlContextRegex = regexp.MustCompile(`(?i)(?:url\s*\(|@import)\s*['"]?\s*$`)

func (c *Crawler) registerHandlers(ctx context.Context, collector *colly.Collector, scope *scope, results chan<- Result) {
	collector.OnRequest(func(request *colly.Request) {
		if ctx.Err() != nil {
			request.Abort()

			return
		}

		if match := c.fileURLsNotToRequestExtRegex.MatchString(path.Ext(strings.ToLower(request.URL.Path))); match {
			request.Abort()
		}
	})

	collector.OnError(func(response *colly.Response, err error) {
		if ctx.Err() != nil {
			return
		}

		// Probing /sitemap.xml.gz on servers that do not actually serve gzip
		// (or return 404 HTML) makes colly fail with gzip.ErrHeader; that is
		// an expected probe outcome, not an error worth surfacing.
		if isNonGzippedXMLGZProbe(response.Request.URL.String(), err) {
			return
		}

		emit(ctx, results, Result{
			Type:   ResultError,
			Source: response.Request.URL.String(),
			Error:  fmt.Errorf("failed to request %s: %w", response.Request.URL.String(), err),
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

		rawPath := strings.ToLower(response.Request.URL.Path)

		switch {
		case isExtractableMediaType(mediatype):
		case c.fileURLsToRequestExtRegex.MatchString(path.Ext(rawPath)):
		case strings.HasSuffix(rawPath, ".xml.gz"): // colly gunzips *.xml.gz responses.
		default:
			return
		}

		isHTML := mediatype == "text/html" || mediatype == "application/xhtml+xml"

		via := "regex"

		if extension := strings.ToLower(strings.TrimPrefix(path.Ext(response.Request.URL.Path), ".")); extension != "" {
			via += ":" + extension
		}

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
			"&amp;", "&",
			"&#038;", "&",
			"&#x26;", "&",
			"&lt;", "<",
			"&gt;", ">",
			"&quot;", `"`,
			"&#039;", "'",
			"&#x27;", "'",
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
			if !strings.Contains(link, "://") {
				if isHTML || !isExtractableRelative(body, loc) {
					continue
				}
			}

			c.processLink(ctx, response.Request, link, via, scope, results)
		}

		if isHTML && c.renderer != nil {
			var renderedHTML string

			renderedHTML, err := c.render(ctx, response.Request.URL.String())
			if err != nil {
				emit(ctx, results, Result{
					Type:   ResultError,
					Source: response.Request.URL.String(),
					Error:  fmt.Errorf("failed to render %s: %w", response.Request.URL.String(), err),
				})

				return
			}

			// Replace the body with the rendered document so colly's HTML
			// parsing (and therefore the OnHTML handlers) runs against the
			// post-JavaScript DOM.
			response.Body = []byte(renderedHTML)
		}
	})

	collector.OnHTML("[href]", func(e *colly.HTMLElement) {
		c.processLink(ctx, e.Request, e.Attr("href"), "html:href", scope, results)
	})

	collector.OnHTML("[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")

		if absoluteURL := e.Request.AbsoluteURL(link); scope.allows(absoluteURL) && strings.Contains(absoluteURL, ".min.") {
			companion := normalizeURL(strings.ReplaceAll(absoluteURL, ".min.", "."))

			c.visit(ctx, companion, results, scope, func() error {
				return e.Request.Visit(companion)
			})
		}

		c.processLink(ctx, e.Request, link, "html:src", scope, results)
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
	absoluteURL := request.AbsoluteURL(link)

	canonicalURL := normalizeURL(absoluteURL)

	if !scope.allows(canonicalURL) || c.alreadySeen(canonicalURL) {
		return
	}

	result := Result{
		Type:   ResultURL,
		Source: request.URL.String(),
		Value:  canonicalURL,
		Via:    via,
	}

	if !emit(ctx, results, result) {
		return
	}

	c.visit(ctx, canonicalURL, results, scope, func() error {
		return request.Visit(canonicalURL)
	})
}

func (c *Crawler) visit(ctx context.Context, rawURL string, results chan<- Result, scope *scope, do func() error) {
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
			Error: fmt.Errorf("failed to visit %s: %w", rawURL, err),
		})
	}
}

func (c *Crawler) alreadySeen(rawURL string) (seen bool) {
	_, seen = c.seen.LoadOrStore(c.dedupeKey(rawURL), struct{}{})

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

func isNonGzippedXMLGZProbe(rawURL string, err error) (expected bool) {
	if !strings.HasSuffix(strings.ToLower(rawURL), ".xml.gz") {
		return
	}

	return errors.Is(err, gzip.ErrHeader)
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

	if !strings.Contains(rawPath, "sitemap") {
		return
	}

	return strings.HasSuffix(rawPath, ".xml") || strings.HasSuffix(rawPath, ".xml.gz")
}

func isExtractableRelative(body string, loc []int) (extractable bool) {
	// A match ending right before ":" is a fragment of a host:port
	// reference (the extractor splits on the port colon), not a usable URL.
	if loc[1] < len(body) && body[loc[1]] == ':' {
		return
	}

	// On other bodies (CSS, JavaScript, JSON, XML), relative matches are
	// accepted only in CSS reference contexts (url(...)/@import); everything
	// else is code, data, or a fragment of an absolute URL.
	start := loc[0]

	if start > 32 {
		start -= 32
	} else {
		start = 0
	}

	extractable = urlContextRegex.MatchString(body[start:loc[0]])

	return
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

const defaultRenderTimeout = 10 * time.Second

type renderer struct {
	timeout time.Duration

	mu          sync.Mutex
	browser     context.Context //nolint:containedctx // browser lifecycle handle, not a request scope.
	cancel      context.CancelFunc
	cancelAlloc context.CancelFunc
	started     atomic.Bool
	err         error
}

func (r *renderer) start() (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started.Load() {
		return r.err
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), chromedp.DefaultExecAllocatorOptions[:]...)

	r.browser, r.cancel = chromedp.NewContext(allocCtx)

	r.err = chromedp.Do(r.browser, chromedp.Evaluate[chromedp.Void]("1"))
	if r.err != nil {
		r.cancel()
		cancelAlloc()

		r.browser = nil
		r.cancel = nil

		return
	}

	r.cancelAlloc = cancelAlloc
	r.started.Store(true)

	return
}

func (r *renderer) close() {
	if !r.started.Load() {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started.Load() {
		return
	}

	r.cancel()
	r.cancelAlloc()

	r.browser = nil
	r.cancel = nil
	r.cancelAlloc = nil
	r.err = nil
	r.started.Store(false)
}

func (c *Crawler) Validate() (err error) {
	if c.renderer == nil {
		return
	}

	err = c.renderer.start()

	return
}

func (c *Crawler) Close() {
	if c.renderer != nil {
		c.renderer.close()
	}
}

// render loads rawURL in a headless Chrome tab and returns the rendered
// document.
func (c *Crawler) render(ctx context.Context, rawURL string) (renderedHTML string, err error) {
	if err = c.renderer.start(); err != nil {
		return
	}

	tabCtx, cancelTab := chromedp.NewContext(c.renderer.browser)
	defer cancelTab()

	tabCtx, cancelTimeout := context.WithTimeout(tabCtx, c.renderer.timeout)
	defer cancelTimeout()

	stop := context.AfterFunc(ctx, cancelTimeout)
	defer stop()

	if err = chromedp.Do(tabCtx, //nolint:contextcheck // tabCtx is a chromedp tab context, derived from the browser context.
		chromedp.Navigate(rawURL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(500*time.Millisecond),
	); err != nil {
		return
	}

	renderedHTML, err = chromedp.Run(tabCtx, chromedp.OuterHTML(chromedp.CSS("html"))) //nolint:contextcheck // tabCtx is a chromedp tab context.

	return
}
