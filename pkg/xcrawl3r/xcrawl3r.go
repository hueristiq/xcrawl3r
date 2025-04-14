package xcrawl3r

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/gocolly/colly/v2/storage"
	"github.com/hueristiq/hq-go-url/extractor"
	"github.com/hueristiq/hq-go-url/parser"
)

type Crawler struct {
	cfg *Configuration

	_URLFilterRegex    *regexp.Regexp
	_URLExtractorRegex *regexp.Regexp

	fileURLsToRequestExtRegex    *regexp.Regexp
	fileURLsNotToRequextExtRegex *regexp.Regexp

	storage *storage.InMemoryStorage
}

func (c *Crawler) Crawl(target string) <-chan Result {
	results := make(chan Result)

	go func() {
		defer close(results)

		targets, err := c.targets(target)
		if err != nil {
			result := Result{
				Type:  ResultError,
				Error: fmt.Errorf("error creating targets for %s: %w", target, err),
			}

			results <- result

			return
		}

		collector, err := c.collector()
		if err != nil {
			result := Result{
				Type:  ResultError,
				Error: fmt.Errorf("error creating collector for %s: %w", target, err),
			}

			results <- result

			return
		}

		isURLToFileContextKey := "isURLToFileContextKey"
		isURLToFileContextTrueValue := "true"
		isURLToFileContextFalseValue := "false"

		collector.OnRequest(func(request *colly.Request) {
			ext := path.Ext(request.URL.Path)

			if match := c.fileURLsNotToRequextExtRegex.MatchString(ext); match {
				request.Abort()

				return
			}

			request.Ctx.Put(isURLToFileContextKey, isURLToFileContextFalseValue)

			if match := c.fileURLsToRequestExtRegex.MatchString(ext); match {
				request.Ctx.Put(isURLToFileContextKey, isURLToFileContextTrueValue)
			}
		})

		collector.OnError(func(response *colly.Response, err error) {
			result := Result{
				Type:  ResultError,
				Error: fmt.Errorf("error requesting %s: %w", response.Request.URL.String(), err),
			}

			results <- result
		})

		collector.OnResponse(func(response *colly.Response) {
			if response.Ctx.Get(isURLToFileContextKey) == isURLToFileContextFalseValue {
				return
			}

			body := string(response.Body)

			replacer := strings.NewReplacer(
				"*", "",
				`\u002f`, "/",
				`\u0026`, "&",
			)

			body = replacer.Replace(body)

			links := c._URLExtractorRegex.FindAllString(body, -1)

			for _, link := range links {
				URL := response.Request.AbsoluteURL(link)

				if valid := c.validate(URL); !valid {
					continue
				}

				result := Result{
					Type:  ResultURL,
					Value: URL,
				}

				results <- result

				if err := response.Request.Visit(URL); err != nil {
					result := Result{
						Type:  ResultError,
						Error: fmt.Errorf("error visiting %s: %w", URL, err),
					}

					results <- result
				}
			}
		})

		collector.OnHTML("[href]", func(e *colly.HTMLElement) {
			if e.Request.Ctx.Get(isURLToFileContextKey) == isURLToFileContextTrueValue {
				return
			}

			link := e.Attr("href")

			URL := e.Request.AbsoluteURL(link)

			if valid := c.validate(URL); !valid {
				return
			}

			result := Result{
				Type:  ResultURL,
				Value: URL,
			}

			results <- result

			if err := e.Request.Visit(URL); err != nil {
				result := Result{
					Type:  ResultError,
					Error: fmt.Errorf("error visiting %s: %w", URL, err),
				}

				results <- result
			}
		})

		collector.OnHTML("[src]", func(e *colly.HTMLElement) {
			if e.Request.Ctx.Get(isURLToFileContextKey) == isURLToFileContextTrueValue {
				return
			}

			link := e.Attr("src")

			URL := e.Request.AbsoluteURL(link)

			if valid := c.validate(URL); !valid {
				return
			}

			result := Result{
				Type:  ResultURL,
				Value: URL,
			}

			results <- result

			if err := e.Request.Visit(URL); err != nil {
				result := Result{
					Type:  ResultError,
					Error: fmt.Errorf("error visiting %s: %w", URL, err),
				}

				results <- result
			}

			if strings.Contains(URL, ".min.") {
				URL = strings.ReplaceAll(URL, ".min.", ".")

				if err := e.Request.Visit(URL); err != nil {
					result := Result{
						Type:  ResultError,
						Error: fmt.Errorf("error visiting %s: %w", URL, err),
					}

					results <- result
				}
			}
		})

		for _, target = range targets {
			if err := collector.Visit(target); err != nil {
				result := Result{
					Type:  ResultError,
					Error: fmt.Errorf("error visiting %s: %w", target, err),
				}

				results <- result
			}
		}

		collector.Wait()
	}()

	return results
}

func (c *Crawler) targets(target string) (targets []string, err error) {
	targets = []string{}

	var parsedTargetURL *parser.URL

	parsedTargetURL, err = up.Parse(target)
	if err != nil {
		return
	}

	targets = append(targets, parsedTargetURL.String())

	if strings.Contains(parsedTargetURL.String(), ".min.") {
		targets = append(targets, strings.ReplaceAll(parsedTargetURL.String(), ".min.", "."))
	}

	robotsTXTURL := fmt.Sprintf("%s://%s/robots.txt", parsedTargetURL.Scheme, parsedTargetURL.Host)

	targets = append(targets, robotsTXTURL)

	sitemaps := []string{
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

	for _, sitemap := range sitemaps {
		sitemapURL := fmt.Sprintf("%s://%s%s", parsedTargetURL.Scheme, parsedTargetURL.Host, sitemap)

		targets = append(targets, sitemapURL)
	}

	return
}

func (c *Crawler) collector() (collector *colly.Collector, err error) {
	collector = colly.NewCollector(
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.URLFilters(c._URLFilterRegex),
		colly.MaxDepth(c.cfg.Depth),
	)

	if err = collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: c.cfg.Parallelism,
		RandomDelay: time.Duration(c.cfg.Delay) * time.Second,
	}); err != nil {
		return
	}

	if len(c.cfg.Headers) > 0 {
		collector.OnRequest(func(request *colly.Request) {
			for _, entry := range c.cfg.Headers {
				var splitEntry []string

				switch {
				case strings.Contains(entry, ": "):
					splitEntry = strings.SplitN(entry, ": ", 2)
				case strings.Contains(entry, ":"):
					splitEntry = strings.SplitN(entry, ":", 2)
				default:
					continue
				}

				header := strings.TrimSpace(splitEntry[0])
				value := splitEntry[1]

				request.Headers.Set(header, value)
			}
		})
	}

	extensions.Referer(collector)

	HTTPTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(c.cfg.Timeout) * time.Second,
			KeepAlive: time.Duration(c.cfg.Timeout) * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     time.Duration(c.cfg.Timeout) * time.Second,
		TLSHandshakeTimeout: time.Duration(c.cfg.Timeout) * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
	}

	HTTPClient := &http.Client{
		Transport: HTTPTransport,
	}

	// NOTE: Must come BEFORE .SetClient calls
	collector.SetClient(HTTPClient)

	// NOTE: Must come AFTER .SetClient calls
	if len(c.cfg.Proxies) > 0 {
		var rrps colly.ProxyFunc

		rrps, err = proxy.RoundRobinProxySwitcher(c.cfg.Proxies...)
		if err != nil {
			return
		}

		collector.SetProxyFunc(rrps)
	}

	if c.cfg.Debug {
		collector.SetDebugger(&debug.LogDebugger{})
	}

	collector.SetStorage(c.storage)

	return
}

func (c *Crawler) validate(URL string) (valid bool) {
	valid = c._URLFilterRegex.MatchString(URL)

	return
}

type Result struct {
	Type  ResultType
	Value string
	Error error
}

type ResultType int

type Configuration struct {
	Domains           []string
	IncludeSubdomains bool
	Delay             int
	Headers           []string
	Timeout           int
	Proxies           []string
	Depth             int
	Parallelism       int
	Debug             bool
}

var (
	up = parser.New(parser.WithDefaultScheme("https"))
)

const (
	ResultURL ResultType = iota
	ResultError
)

func New(cfg *Configuration) (crawler *Crawler, err error) {
	crawler = &Crawler{
		cfg: cfg,
	}

	URLFilterRegexPattern := `https?://([a-z0-9-]+\.)(?:[a-z0-9-]+\.)+[a-z]{2,}(:\d+)?(?:/[^?\s#]*)?(?:\?[^#\s]*)?(?:#[^\s]*)?`

	if cfg.Domains != nil {
		var b strings.Builder

		b.WriteString("(?:")

		for i, s := range cfg.Domains {
			if i != 0 {
				b.WriteByte('|')
			}

			b.WriteString(regexp.QuoteMeta(s))
		}

		b.WriteByte(')')

		URLFilterRegexPattern = fmt.Sprintf(`https?://(www\.)?%s(:\d+)?(?:/[^?\s#]*)?(?:\?[^#\s]*)?(?:#[^\s]*)?`, b.String())

		if cfg.IncludeSubdomains {
			URLFilterRegexPattern = fmt.Sprintf(`https?://([a-z0-9-]+\.)*%s(:\d+)?(?:/[^?\s#]*)?(?:\?[^#\s]*)?(?:#[^\s]*)?`, b.String())
		}
	}

	crawler._URLFilterRegex = regexp.MustCompile(URLFilterRegexPattern)
	crawler._URLExtractorRegex = extractor.New().CompileRegex()

	crawler.fileURLsToRequestExtRegex = regexp.MustCompile(`\.(css|csv|js|json|map|txt|xml|yaml|yml)$`)
	crawler.fileURLsNotToRequextExtRegex = regexp.MustCompile(`\.(apng|bpm|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf)$`)

	crawler.storage = &storage.InMemoryStorage{}

	return
}
