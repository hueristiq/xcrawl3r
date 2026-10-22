package xcrawl3r

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
	hqgourlextractor "github.com/hueristiq/hq-lib-url-go/extractor"
)

// scriptBlockRegex captures the content of inline <script> blocks in HTML.
var scriptBlockRegex = regexp.MustCompile(`(?is)<script\b[^>]*>(.*?)</script>`)

// knownPathExtensions lists the file extensions that make a dot-containing
// relative reference look like a path. Ported from jsluice's MaybeURL.
var knownPathExtensions = map[string]struct{}{
	"asp":   {},
	"ashx":  {},
	"asmx":  {},
	"aspx":  {},
	"cgi":   {},
	"css":   {},
	"do":    {},
	"doc":   {},
	"docx":  {},
	"htm":   {},
	"html":  {},
	"js":    {},
	"json":  {},
	"jsonp": {},
	"jsp":   {},
	"jspa":  {},
	"map":   {},
	"pdf":   {},
	"php":   {},
	"phtml": {},
	"pl":    {},
	"py":    {},
	"rb":    {},
	"rss":   {},
	"txt":   {},
	"xhtml": {},
	"xls":   {},
	"xlsx":  {},
	"xml":   {},
}

// maybePath reports whether a relative match looks like a URL path rather
// than code or data. The checks mirror jsluice's MaybeURL: a leading slash,
// a dotted host, a query value, or a known file extension.
func maybePath(link string) (path bool) {
	if link == "/" || !strings.ContainsAny(link, "/?.") {
		return
	}

	if strings.HasPrefix(link, "/") {
		return true
	}

	parsedURL, err := url.Parse(link)
	if err != nil {
		return
	}

	if parsedURL.Scheme != "" {
		scheme := strings.ToLower(parsedURL.Scheme)

		if scheme != schemeHTTP && scheme != schemeHTTPS {
			return
		}
	}

	if len(strings.Split(parsedURL.Hostname(), ".")) > 1 {
		return true
	}

	for _, values := range parsedURL.Query() {
		if len(values) > 0 && values[0] != "" {
			return true
		}
	}

	if !strings.ContainsAny(parsedURL.Path, ".") {
		return
	}

	parts := strings.Split(parsedURL.Path, ".")
	extension := strings.ToLower(parts[len(parts)-1])

	_, path = knownPathExtensions[extension]

	return
}

// handleJSLiterals finds string-literal concatenation groups in body,
// collapses them, and extracts links from the collapsed text. Single
// literals without a known API context are skipped: the body scans already
// cover them.
func (c *Crawler) handleJSLiterals(ctx context.Context, request *colly.Request, body string, scope *scope, results chan<- Result) {
	c.handleJSLiteralsFrom(ctx, request.URL, request.Visit, body, scope, results)
}

// handleJSLiteralsFrom is handleJSLiterals with an explicit base URL.
func (c *Crawler) handleJSLiteralsFrom(ctx context.Context, baseURL *url.URL, visit visitFunc, body string, scope *scope, results chan<- Result) {
	for _, group := range hqgourlextractor.JSLiteralGroups(body) {
		via := jsContextTag(body, group.Start)

		if via == "" && len(group.Pieces) == 1 && !group.HasExpr {
			continue
		}

		if via == "" {
			via = "js:concat"
		}

		collapsed := group.Collapsed()

		for _, loc := range c.urlExtractorRegex.FindAllStringIndex(collapsed, -1) {
			link := collapsed[loc[0]:loc[1]]

			// A trailing EXPR segment or query value is unknown-but-optional:
			// trim it so the rest of the URL is fetchable.
			link = strings.TrimSuffix(link, "/"+hqgourlextractor.ExpressionPlaceholder)
			link = strings.TrimSuffix(link, "="+hqgourlextractor.ExpressionPlaceholder)

			if link == "" || link == "/" || link == "//" {
				continue
			}

			if !c.usableLink(collapsed, loc, link, bodyScanOptions{allowRelative: true}) {
				continue
			}

			// A link that still contains the EXPR placeholder cannot be
			// fetched as it is; record it without queueing a visit.
			if strings.Contains(link, hqgourlextractor.ExpressionPlaceholder) {
				c.emitLinkFrom(ctx, baseURL, link, via, scope, results)

				continue
			}

			c.processLinkFrom(ctx, baseURL, visit, link, via, scope, results)
		}
	}
}

// handleJSSnippet scans a JavaScript snippet (an event-handler attribute
// value or a sourcemap sourcesContent entry) for links: the collapsed
// literal groups, the raw text, and its escape-decoded variant. Relative
// links are resolved against the request URL.
func (c *Crawler) handleJSSnippet(ctx context.Context, request *colly.Request, snippet, via string, scope *scope, results chan<- Result) {
	c.handleJSSnippetFrom(ctx, request.URL, request.Visit, snippet, via, scope, results)
}

// handleJSSnippetFrom is handleJSSnippet with an explicit base URL.
func (c *Crawler) handleJSSnippetFrom(ctx context.Context, baseURL *url.URL, visit visitFunc, snippet, via string, scope *scope, results chan<- Result) {
	c.handleJSLiteralsFrom(ctx, baseURL, visit, snippet, scope, results)

	c.scanBodyFrom(ctx, baseURL, visit, snippet, via, bodyScanOptions{allowRelative: true}, scope, results)
	c.scanBodyFrom(ctx, baseURL, visit, hqgourlextractor.DecodeJSEscapes(snippet), via, bodyScanOptions{allowRelative: true}, scope, results)
}

// jsContextWindow bounds how far back before a literal group the API context
// is looked for.
const jsContextWindow = 120

// jsContextPattern associates a trailing-source pattern with the via tag of
// the JavaScript API call or assignment it describes.
type jsContextPattern struct {
	re  *regexp.Regexp
	via string
}

// jsContextPatterns lists the JavaScript API contexts recognized for a string
// literal. Patterns are anchored at the end of the window preceding the
// literal, so the literal must be the call's (first) URL argument. They
// mirror jsluice's URL matchers for the common cases.
var jsContextPatterns = []jsContextPattern{
	{regexp.MustCompile(`\.open\s*\(\s*["']?(?:GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)["']?\s*,\s*$`), "js:xhr.open"},
	{regexp.MustCompile(`fetch\s*\(\s*$`), "js:fetch"},
	{regexp.MustCompile(`(?:window\.)?open\s*\(\s*$`), "js:window.open"},
	{regexp.MustCompile(`(?:window\.)?location\s*(?:\.href\s*=\s*|=\s*|\.(?:replace|assign)\s*\(\s*)$`), "js:location"},
	{regexp.MustCompile(`sendBeacon\s*\(\s*$`), "js:sendbeacon"},
	{regexp.MustCompile(`new\s+WebSocket\s*\(\s*$`), "js:websocket"},
	{regexp.MustCompile(`new\s+EventSource\s*\(\s*$`), "js:eventsource"},
	{regexp.MustCompile(`import\s*\(\s*$`), "js:import"},
	{regexp.MustCompile(`\$(?:\.\w+)?\.(?:get|post|ajax)\s*\(\s*$`), "js:jquery"},
	{regexp.MustCompile(`axios(?:\.\w+)?\s*\(\s*$`), "js:axios"},
}

// jsContextTag reports the via tag of the JavaScript API context a literal
// group starting at start belongs to, or "" when none matches.
func jsContextTag(body string, start int) (tag string) {
	from := start - jsContextWindow

	if from < 0 {
		from = 0
	}

	window := body[from:start]

	for _, pattern := range jsContextPatterns {
		if pattern.re.MatchString(window) {
			return pattern.via
		}
	}

	return
}
