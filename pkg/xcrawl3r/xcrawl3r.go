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

	_URLFilterRegex    *regexp.Regexp
	_URLExtractorRegex *regexp.Regexp

	fileURLsToRequestExtRegex    *regexp.Regexp
	fileURLsNotToRequextExtRegex *regexp.Regexp

	pageCollectorStorage *storage.InMemoryStorage
	fileCollectorStorage *storage.InMemoryStorage
}

func (crawler *Crawler) Crawl(target string) <-chan Result {
	results := make(chan Result)

	go func() {
		defer close(results)

		targets, err := crawler.targets(target)
		if err != nil {
			result := Result{
				Type:   ResultError,
				Source: "page:href",
				Error:  err,
			}

			results <- result

			return
		}

		p, f, err := crawler.collectors()
		if err != nil {
			result := Result{
				Type:   ResultError,
				Source: "page:href",
				Error:  err,
			}

			results <- result

			return
		}

		wg := &sync.WaitGroup{}

		p.OnRequest(func(request *colly.Request) {
			ext := path.Ext(request.URL.Path)

			if match := crawler.fileURLsNotToRequextExtRegex.MatchString(ext); match {
				request.Abort()

				wg.Done()

				return
			}

			if match := crawler.fileURLsToRequestExtRegex.MatchString(ext); match {
				wg.Add(1)

				if err := f.Visit(request.URL.String()); err != nil {
					result := Result{
						Type:   ResultError,
						Source: "page:href",
						Error:  err,
					}

					results <- result

					wg.Done()
				}

				request.Abort()

				wg.Done()

				return
			}
		})

		p.OnError(func(_ *colly.Response, err error) {
			result := Result{
				Type:   ResultError,
				Source: "page",
				Error:  err,
			}

			results <- result

			wg.Done()
		})

		p.OnResponse(func(response *colly.Response) {
			wg.Done()
		})

		p.OnHTML("[href]", func(e *colly.HTMLElement) {
			link := e.Attr("href")

			URL := e.Request.AbsoluteURL(link)

			if valid := crawler.validate(URL); !valid {
				return
			}

			result := Result{
				Type:   ResultURL,
				Source: "page:href",
				Value:  URL,
			}

			results <- result

			wg.Add(1)

			if err := e.Request.Visit(URL); err != nil {
				result := Result{
					Type:   ResultError,
					Source: "page:href",
					Error:  err,
				}

				results <- result

				wg.Done()
			}
		})

		p.OnHTML("[src]", func(e *colly.HTMLElement) {
			link := e.Attr("src")

			URL := e.Request.AbsoluteURL(link)

			if valid := crawler.validate(URL); !valid {
				return
			}

			result := Result{
				Type:   ResultURL,
				Source: "page:src",
				Value:  URL,
			}

			results <- result

			wg.Add(1)

			if err := e.Request.Visit(URL); err != nil {
				result := Result{
					Type:   ResultError,
					Source: "page:href",
					Error:  err,
				}

				results <- result

				wg.Done()
			}
		})

		f.OnRequest(func(request *colly.Request) {
			if strings.Contains(request.URL.String(), ".min.") {
				wg.Add(1)

				if err := f.Visit(strings.ReplaceAll(request.URL.String(), ".min.", ".")); err != nil {
					result := Result{
						Type:   ResultError,
						Source: "page:href",
						Error:  err,
					}

					results <- result

					wg.Done()
				}
			}
		})

		f.OnError(func(_ *colly.Response, err error) {
			result := Result{
				Type:   ResultError,
				Source: "page",
				Error:  err,
			}

			results <- result

			wg.Done()
		})

		f.OnResponse(func(response *colly.Response) {
			ext := path.Ext(response.Request.URL.Path)

			body := string(response.Body)

			replacer := strings.NewReplacer(
				"*", "",
				`\u002f`, "/",
				`\u0026`, "&",
			)

			body = replacer.Replace(body)

			links := crawler._URLExtractorRegex.FindAllString(body, -1)

			for _, link := range links {
				URL := response.Request.AbsoluteURL(link)

				if valid := crawler.validate(URL); !valid {
					continue
				}

				result := Result{
					Type:   ResultURL,
					Source: "file:" + ext,
					Value:  URL,
				}

				results <- result

				wg.Add(1)

				if err := p.Visit(URL); err != nil {
					result := Result{
						Type:   ResultError,
						Source: "page:href",
						Error:  err,
					}

					results <- result

					wg.Done()
				}
			}

			wg.Done()
		})

		for i := range targets {
			wg.Add(1)

			if err := p.Visit(targets[i]); err != nil {
				result := Result{
					Type:   ResultError,
					Source: "page:href",
					Error:  err,
				}

				results <- result

				wg.Done()
			}
		}

		p.Wait()
		f.Wait()
		wg.Wait()
	}()

	return results
}

func (crawler *Crawler) targets(target string) (targets []string, err error) {
	targets = []string{}

	var parsedTargetURL *parser.URL

	parsedTargetURL, err = up.Parse(target)
	if err != nil {
		return
	}

	targets = append(targets, parsedTargetURL.String())

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

func (crawler *Crawler) collectors() (p, f *colly.Collector, err error) {
	p = colly.NewCollector(
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.URLFilters(crawler._URLFilterRegex),
		colly.MaxDepth(crawler.cfg.Depth),
	)

	if err = p.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: crawler.cfg.Parallelism,
		RandomDelay: time.Duration(crawler.cfg.Delay) * time.Second,
	}); err != nil {
		return
	}

	if len(crawler.cfg.Headers) > 0 {
		p.OnRequest(func(request *colly.Request) {
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

	extensions.Referer(p)

	HTTPTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(crawler.cfg.Timeout) * time.Second,
			KeepAlive: time.Duration(crawler.cfg.Timeout) * time.Second,
		}).DialContext,
		MaxIdleConns:        100, // Golang default is 100
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     time.Duration(crawler.cfg.Timeout) * time.Second,
		TLSHandshakeTimeout: time.Duration(crawler.cfg.Timeout) * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
	}

	HTTPClient := &http.Client{
		Transport: HTTPTransport,
	}

	// NOTE: Must come BEFORE .SetClient calls
	p.SetClient(HTTPClient)

	// Proxies
	// NOTE: Must come AFTER .SetClient calls
	if len(crawler.cfg.Proxies) > 0 {
		var rrps colly.ProxyFunc

		rrps, err = proxy.RoundRobinProxySwitcher(crawler.cfg.Proxies...)
		if err != nil {
			return
		}

		p.SetProxyFunc(rrps)
	}

	if crawler.cfg.Debug {
		p.SetDebugger(&debug.LogDebugger{})
	}

	f = p.Clone()
	f.URLFilters = nil

	p.ID = 1
	p.SetStorage(crawler.pageCollectorStorage)

	f.ID = 2
	f.SetStorage(crawler.fileCollectorStorage)

	return
}

func (crawler *Crawler) validate(URL string) (valid bool) {
	valid = crawler._URLFilterRegex.MatchString(URL)

	return
}

type Configuration struct {
	Domains           []string
	IncludeSubdomains bool
	Depth             int
	Parallelism       int
	Delay             int
	Headers           []string
	Timeout           int
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

	crawler.fileURLsToRequestExtRegex = regexp.MustCompile(`\.(css|js|json|xml|csv|txt)$`)
	crawler.fileURLsNotToRequextExtRegex = regexp.MustCompile(`\.(apng|bpm|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf)$`)

	crawler.pageCollectorStorage = &storage.InMemoryStorage{}
	crawler.fileCollectorStorage = &storage.InMemoryStorage{}

	return
}
