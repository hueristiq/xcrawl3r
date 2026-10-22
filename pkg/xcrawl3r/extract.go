package xcrawl3r

import (
	"context"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
)

// urlContextRegex matches the text immediately before a relative reference
// inside a CSS url(...) or @import context, e.g. "url(", "url('", `url("`,
// or `@import "`.
var urlContextRegex = regexp.MustCompile(`(?i)(?:url\s*\(|@import)\s*['"]?\s*$`)

const (
	mediaTypeHTML       = "text/html"
	mediaTypeXHTML      = "application/xhtml+xml"
	mediaTypeJSON       = "application/json"
	mediaTypeJavaScript = "application/javascript"
	mediaTypeXML        = "application/xml"
	mediaTypeForm       = "application/x-www-form-urlencoded"
)

type bodyScanOptions struct {
	// allowRelative accepts relative matches that pass maybePath; otherwise
	// only CSS url(...)/@import contexts are accepted.
	allowRelative bool
}

// scanBody runs the URL extractor over body and processes every usable
// match.
func (c *Crawler) scanBody(ctx context.Context, request *colly.Request, body, via string, opts bodyScanOptions, scope *scope, results chan<- Result) {
	c.scanBodyFrom(ctx, request.URL, request.Visit, body, via, opts, scope, results)
}

// scanBodyFrom is scanBody with an explicit base URL, used when scanning
// content that belongs to a URL other than the one that was requested.
func (c *Crawler) scanBodyFrom(ctx context.Context, baseURL *url.URL, visit visitFunc, body, via string, opts bodyScanOptions, scope *scope, results chan<- Result) {
	for _, loc := range c.urlExtractorRegex.FindAllStringIndex(body, -1) {
		link := body[loc[0]:loc[1]]

		if !c.usableLink(body, loc, link, opts) {
			continue
		}

		c.processLinkFrom(ctx, baseURL, visit, link, via, scope, results)
	}
}

// usableLink reports whether the match at loc is worth extracting. Relative
// matches are guarded against HTML tag tails and JS line comments, then
// filtered by context (maybePath or CSS references).
func (c *Crawler) usableLink(body string, loc []int, link string, opts bodyScanOptions) (usable bool) {
	relative := !strings.Contains(link, "://")

	if relative && loc[0] > 0 {
		switch body[loc[0]-1] {
		case '<':
			// The tail of an HTML/XML tag ("/div" in "</div>").
			return
		case '/':
			// The second slash of a "//" line comment.
			return
		}
	}

	if !relative {
		return true
	}

	// "//host/path" is scheme-relative, but "//comment" is a line comment;
	// only treat it as a link when the remainder looks like a host.
	if strings.HasPrefix(link, "//") {
		rest := strings.TrimPrefix(link, "//")

		if slash := strings.IndexByte(rest, '/'); slash > 0 {
			rest = rest[:slash]
		}

		return strings.Contains(rest, ".")
	}

	if opts.allowRelative {
		return maybePath(link)
	}

	return isExtractableRelative(body, loc)
}

func isExtractableMediaType(mediatype string) (extractable bool) {
	if strings.HasPrefix(mediatype, "text/") {
		return true
	}

	switch mediatype {
	case mediaTypeJSON,
		mediaTypeJavaScript,
		mediaTypeXML,
		mediaTypeXHTML,
		mediaTypeForm:
		return true
	}

	return strings.HasSuffix(mediatype, "+json") || strings.HasSuffix(mediatype, "+xml")
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

// isJSBody reports whether a response body is JavaScript or JSON, and so
// worth scanning for relative links and string-literal concatenations.
func isJSBody(mediatype, rawPath string) (js bool) {
	if strings.Contains(mediatype, "javascript") {
		return true
	}

	if strings.HasSuffix(mediatype, "json") {
		return true
	}

	switch path.Ext(rawPath) {
	case ".js", ".mjs", ".cjs", ".json", ".map":
		return true
	}

	return
}

// isJSCSSBody reports whether a response body is JavaScript, JSON, or CSS, and
// so may carry a sourceMappingURL directive.
func isJSCSSBody(mediatype, rawPath string) (jscss bool) {
	if isJSBody(mediatype, rawPath) {
		return true
	}

	if strings.Contains(mediatype, "css") {
		return true
	}

	return strings.HasSuffix(rawPath, ".css")
}
