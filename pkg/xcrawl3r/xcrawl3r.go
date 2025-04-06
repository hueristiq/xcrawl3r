package xcrawl3r

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
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

	URLsExtractorRegex    *regexp.Regexp
	URLFilterRegex        *regexp.Regexp
	URLsNotToRequestRegex *regexp.Regexp
	URLsToFilesRegex      *regexp.Regexp
	PageCollector         *colly.Collector
	FileCollector         *colly.Collector
}

func (crawler *Crawler) Crawl(targetURL string) <-chan Result {
	results := make(chan Result)

	parsedTargetURL, err := up.Parse(targetURL)
	if err != nil {
		result := Result{
			Type:   ResultError,
			Source: "page:href",
			Error:  err,
		}

		results <- result

		close(results)

		return results
	}

	targetURLs := []string{
		parsedTargetURL.String(),
	}

	robotsTXTURL := fmt.Sprintf("%s://%s/robots.txt", parsedTargetURL.Scheme, parsedTargetURL.Host)

	targetURLs = append(targetURLs, robotsTXTURL)

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

		targetURLs = append(targetURLs, sitemapURL)
	}

	go func() {
		defer close(results)

		seenURLs := &sync.Map{}

		crawler.PageCollector.OnRequest(func(request *colly.Request) {
			ext := path.Ext(request.URL.Path)

			if match := crawler.URLsNotToRequestRegex.MatchString(ext); match {
				request.Abort()

				return
			}

			if match := crawler.URLsToFilesRegex.MatchString(ext); match {
				crawler.FileCollector.Visit(request.URL.String())

				request.Abort()

				return
			}
		})

		crawler.PageCollector.OnError(func(_ *colly.Response, err error) {
			result := Result{
				Type:   ResultError,
				Source: "page",
				Error:  err,
			}

			results <- result
		})

		crawler.PageCollector.OnHTML("[href]", func(e *colly.HTMLElement) {
			link := e.Attr("href")

			URL := e.Request.AbsoluteURL(link)

			if valid := crawler.Validate(URL); !valid {
				return
			}

			_, loaded := seenURLs.LoadOrStore(URL, struct{}{})
			if loaded {
				return
			}

			result := Result{
				Type:   ResultURL,
				Source: "page:href",
				Value:  URL,
			}

			results <- result

			e.Request.Visit(URL)
		})

		crawler.PageCollector.OnHTML("[src]", func(e *colly.HTMLElement) {
			link := e.Attr("src")

			URL := e.Request.AbsoluteURL(link)

			if valid := crawler.Validate(URL); !valid {
				return
			}

			_, loaded := seenURLs.LoadOrStore(URL, struct{}{})
			if loaded {
				return
			}

			result := Result{
				Type:   ResultURL,
				Source: "page:src",
				Value:  URL,
			}

			results <- result

			e.Request.Visit(URL)
		})

		crawler.FileCollector.OnRequest(func(request *colly.Request) {
			if strings.Contains(request.URL.String(), ".min.") {
				crawler.FileCollector.Visit(strings.ReplaceAll(request.URL.String(), ".min.", "."))
			}
		})

		crawler.FileCollector.OnError(func(_ *colly.Response, err error) {
			result := Result{
				Type:   ResultError,
				Source: "page",
				Error:  err,
			}

			results <- result
		})

		crawler.FileCollector.OnResponse(func(response *colly.Response) {
			ext := path.Ext(response.Request.URL.Path)

			body := string(response.Body)

			replacer := strings.NewReplacer(
				"*", "",
				`\u002f`, "/",
				`\u0026`, "&",
			)

			body = replacer.Replace(body)

			links := crawler.URLsExtractorRegex.FindAllString(body, -1)

			for _, link := range links {
				URL := response.Request.AbsoluteURL(link)

				if valid := crawler.Validate(URL); !valid {
					continue
				}

				_, loaded := seenURLs.LoadOrStore(URL, struct{}{})
				if loaded {
					continue
				}

				result := Result{
					Type:   ResultURL,
					Source: "file:" + ext,
					Value:  URL,
				}

				results <- result

				crawler.PageCollector.Visit(URL)
			}
		})

		for i := range targetURLs {
			crawler.PageCollector.Visit(targetURLs[i])
		}

		crawler.PageCollector.Wait()
	}()

	return results
}

func (crawler *Crawler) Validate(URL string) (valid bool) {
	valid = crawler.URLFilterRegex.MatchString(URL)

	return
}

type Configuration struct {
	Domains           []string
	IncludeSubdomains bool
	Depth             int
	Parallelism       int
	Delay             int // seconds
	Headers           []string
	Timeout           int // seconds
	Proxies           []string
	Debug             bool
}

var (
	up = parser.New(parser.WithDefaultScheme("https"))
)

func New(cfg *Configuration) (crawler *Crawler, err error) {
	crawler = &Crawler{
		cfg: cfg,
	}

	crawler.URLsExtractorRegex = extractor.New().CompileRegex()

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

	crawler.URLFilterRegex = regexp.MustCompile(URLFilterRegexPattern)
	crawler.URLsNotToRequestRegex = regexp.MustCompile(`\.(apng|bpm|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf)$`)
	crawler.URLsToFilesRegex = regexp.MustCompile(`\.(css|js|json|xml|csv|txt)$`)

	crawler.PageCollector = colly.NewCollector(
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.URLFilters(crawler.URLFilterRegex),
		colly.MaxDepth(cfg.Depth),
	)

	if err = crawler.PageCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: cfg.Parallelism,
		RandomDelay: time.Duration(cfg.Delay) * time.Second,
	}); err != nil {
		return
	}

	if len(crawler.cfg.Headers) > 0 {
		crawler.PageCollector.OnRequest(func(request *colly.Request) {
			for index := range crawler.cfg.Headers {
				entry := crawler.cfg.Headers[index]

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

	extensions.Referer(crawler.PageCollector)

	HTTPTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(cfg.Timeout) * time.Second,
			KeepAlive: time.Duration(cfg.Timeout) * time.Second,
		}).DialContext,
		MaxIdleConns:        100, // Golang default is 100
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     time.Duration(cfg.Timeout) * time.Second,
		TLSHandshakeTimeout: time.Duration(cfg.Timeout) * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
	}

	HTTPClient := &http.Client{
		Transport: HTTPTransport,
	}

	// NOTE: Must come BEFORE .SetClient calls
	crawler.PageCollector.SetClient(HTTPClient)

	// Proxies
	// NOTE: Must come AFTER .SetClient calls
	if len(crawler.cfg.Proxies) > 0 {
		var rrps colly.ProxyFunc

		rrps, err = proxy.RoundRobinProxySwitcher(crawler.cfg.Proxies...)
		if err != nil {
			return
		}

		crawler.PageCollector.SetProxyFunc(rrps)
	}

	if crawler.cfg.Debug {
		crawler.PageCollector.SetDebugger(&debug.LogDebugger{})
	}

	crawler.FileCollector = crawler.PageCollector.Clone()
	crawler.FileCollector.URLFilters = nil

	crawler.PageCollector.ID = 1
	crawler.PageCollector.SetStorage(&storage.InMemoryStorage{})

	crawler.FileCollector.ID = 2
	crawler.FileCollector.SetStorage(&storage.InMemoryStorage{})

	return
}
