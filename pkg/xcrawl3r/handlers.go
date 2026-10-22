package xcrawl3r

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/gocolly/colly/v2"
	hqgourlextractor "github.com/hueristiq/hq-lib-url-go/extractor"
)

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

		isHTML := mediatype == mediaTypeHTML || mediatype == mediaTypeXHTML

		// Only real HTML pages consume the page budget. robots.txt, sitemap
		// probes, *.xml.gz probes, and non-HTML assets are free.
		if isHTML && !isSitemapPath(response.Request.URL.Path) && !strings.HasSuffix(rawPath, ".xml.gz") {
			if !scope.chargePage() {
				// Clearing the body keeps colly's OnHTML handlers from
				// parsing links out of an over-budget page.
				response.Body = nil

				return
			}
		}

		via := "regex"

		if extension := strings.ToLower(strings.TrimPrefix(path.Ext(response.Request.URL.Path), ".")); extension != "" {
			via += ":" + extension
		}

		if isSitemapPath(response.Request.URL.Path) {
			via = "sitemap"
		}

		replacer := strings.NewReplacer(
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

		allowRelative := isJSBody(mediatype, rawPath)

		if isJSCSSBody(mediatype, rawPath) {
			for _, match := range sourceMappingURLRegex.FindAllStringSubmatch(body, -1) {
				c.processLink(ctx, response.Request, match[1], "sourcemap:reference", scope, results)
			}
		}

		if isSourceMapBody(rawPath, mediatype, response.Body) {
			c.handleSourceMap(ctx, response.Request.URL, response.Request.Visit, response.Body, scope, results)
		}

		c.scanBody(ctx, response.Request, body, via, bodyScanOptions{allowRelative: allowRelative}, scope, results)
		c.handleJSLiterals(ctx, response.Request, body, scope, results)
		c.scanBody(ctx, response.Request, hqgourlextractor.DecodeJSEscapes(body), via, bodyScanOptions{allowRelative: allowRelative}, scope, results)

		if isHTML {
			for _, block := range scriptBlockRegex.FindAllStringSubmatch(body, -1) {
				c.handleJSSnippet(ctx, response.Request, block[1], "regex:script", scope, results)
			}
		}

		if isHTML && c.renderer != nil {
			var renderedHTML string

			renderedHTML, err := c.render(ctx, response.Request.URL.String(), scope)
			if err != nil {
				emit(ctx, results, Result{
					Type:   ResultError,
					Source: response.Request.URL.String(),
					Error:  fmt.Errorf("failed to render %s: %w", response.Request.URL.String(), err),
				})

				return
			}

			for _, block := range scriptBlockRegex.FindAllStringSubmatch(renderedHTML, -1) {
				c.handleJSSnippet(ctx, response.Request, block[1], "regex:rendered", scope, results)
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

		if link != "" {
			if companion, ok := minifiedCompanion(e.Request.AbsoluteURL(link)); ok && scope.allows(companion) {
				c.visit(ctx, companion, results, func() error {
					return e.Request.Visit(companion)
				})
			}
		}

		c.processLink(ctx, e.Request, link, "html:src", scope, results)
	})

	collector.OnHTML(`meta[http-equiv="refresh"]`, func(e *colly.HTMLElement) {
		link := metaRefreshURL(e.Attr("content"))

		if link == "" {
			return
		}

		c.processLink(ctx, e.Request, link, "html:meta-refresh", scope, results)
	})

	collector.OnHTML(`meta[property="og:url"], meta[name="twitter:url"]`, func(e *colly.HTMLElement) {
		c.processLink(ctx, e.Request, e.Attr("content"), "html:meta-url", scope, results)
	})

	collector.OnHTML(`link[rel="canonical"]`, func(e *colly.HTMLElement) {
		c.processLink(ctx, e.Request, e.Attr("href"), "html:canonical", scope, results)
	})

	collector.OnHTML("form[action]", func(e *colly.HTMLElement) {
		c.processLink(ctx, e.Request, e.Attr("action"), "html:form", scope, results)
	})

	collector.OnHTML("[srcset]", func(e *colly.HTMLElement) {
		for _, link := range srcsetURLs(e.Attr("srcset")) {
			c.processLink(ctx, e.Request, link, "html:srcset", scope, results)
		}
	})

	collector.OnHTML("video[poster]", func(e *colly.HTMLElement) {
		c.processLink(ctx, e.Request, e.Attr("poster"), "html:poster", scope, results)
	})

	collector.OnHTML(eventHandlerSelector(), func(e *colly.HTMLElement) {
		for _, attribute := range eventHandlerAttributes {
			value := e.DOM.AttrOr(attribute, "")

			if value == "" {
				continue
			}

			c.handleJSSnippet(ctx, e.Request, value, "js:on-"+attribute, scope, results)
		}
	})
}

// metaRefreshURL extracts the URL from a <meta http-equiv="refresh"> content
// value like "5; url=/redirect", or "" when none is present.
func metaRefreshURL(content string) (rawURL string) {
	index := strings.Index(strings.ToLower(content), "url=")

	if index < 0 {
		return
	}

	rawURL = content[index+len("url="):]
	rawURL = strings.TrimSpace(rawURL)
	rawURL = strings.Trim(rawURL, `"'`)

	if space := strings.IndexByte(rawURL, ' '); space >= 0 {
		rawURL = rawURL[:space]
	}

	return
}

// srcsetURLs returns the URL candidates of a srcset attribute value, dropping
// the descriptor ("1x", "640w") that follows each candidate.
func srcsetURLs(value string) (urls []string) {
	for _, candidate := range strings.Split(value, ",") {
		fields := strings.Fields(strings.TrimSpace(candidate))

		if len(fields) > 0 {
			urls = append(urls, fields[0])
		}
	}

	return
}

// eventHandlerAttributes lists the inline event-handler attributes whose
// values are scanned as JavaScript.
var eventHandlerAttributes = []string{
	"onblur",
	"onchange",
	"onclick",
	"ondblclick",
	"onerror",
	"onfocus",
	"oninput",
	"onkeydown",
	"onkeyup",
	"onload",
	"onmouseout",
	"onmouseover",
	"onsubmit",
}

// eventHandlerSelector returns the goquery selector matching any element
// carrying an inline event handler.
func eventHandlerSelector() (selector string) {
	parts := make([]string, 0, len(eventHandlerAttributes))

	for _, attribute := range eventHandlerAttributes {
		parts = append(parts, "["+attribute+"]")
	}

	selector = strings.Join(parts, ",")

	return
}
