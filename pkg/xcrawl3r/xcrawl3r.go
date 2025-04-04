package xcrawl3r

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/hueristiq/xcrawl3r/internal/configuration"
	"go.source.hueristiq.com/url/extractor"
	"go.source.hueristiq.com/url/parser"
)

type Crawler struct {
	FileURLsRegex *regexp.Regexp

	URLsNotToRequestRegex *regexp.Regexp
	URLsRegex             *regexp.Regexp

	PageCollector *colly.Collector
	FileCollector *colly.Collector

	cfg *Configuration
}

func (crawler *Crawler) Crawl(URL string) (results chan Result) {
	results = make(chan Result)

	go func() {
		defer close(results)

		seenURLs := &sync.Map{}

		wg := &sync.WaitGroup{}

		parsedURL, _ := up.Parse(URL)

		wg.Add(1)

		go func() {
			defer wg.Done()

			for result := range crawler.sitemapParsing(parsedURL) {
				_, loaded := seenURLs.LoadOrStore(result.Value, struct{}{})
				if loaded {
					continue
				}

				results <- result
			}
		}()

		wg.Add(1)

		go func() {
			defer wg.Done()

			for result := range crawler.robotsParsing(parsedURL) {
				_, loaded := seenURLs.LoadOrStore(result, struct{}{})
				if loaded {
					continue
				}

				results <- result
			}
		}()

		wg.Add(1)

		go func() {
			defer wg.Done()

			for result := range crawler.pageCrawl(parsedURL) {
				_, loaded := seenURLs.LoadOrStore(result, struct{}{})
				if loaded {
					continue
				}

				results <- result
			}
		}()

		wg.Wait()
	}()

	return
}

type Configuration struct {
	Depth int

	Domain            string
	IncludeSubdomains bool

	Headless  bool
	Headers   []string
	Proxies   []string
	Render    bool
	Timeout   int // seconds
	UserAgent string

	Concurrency    int
	Delay          int // seconds
	MaxRandomDelay int // seconds
	Parallelism    int

	Debug bool

	cfg *Configuration
}

var (
	DefaultUserAgent = fmt.Sprintf("%s v%s (https://github.com/hueristiq/%s)", configuration.NAME, configuration.VERSION, configuration.NAME)
	up               = parser.New(parser.WithDefaultScheme("https"))
)

func New(cfg *Configuration) (crawler *Crawler, err error) {
	crawler = &Crawler{
		cfg: cfg,
	}

	crawler.URLsRegex = extractor.New().CompileRegex()

	crawler.FileURLsRegex = regexp.MustCompile(`(?m).*?\.*(js|json|xml|csv|txt|map)(\?.*?|)$`)

	crawler.URLsNotToRequestRegex = regexp.MustCompile(`(?i)\.(apng|bpm|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf|css)(?:\?|#|$)`)

	crawler.PageCollector = colly.NewCollector(
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.MaxDepth(cfg.Depth),
		colly.AllowedDomains(crawler.cfg.Domain, "www."+crawler.cfg.Domain),
	)

	if crawler.cfg.IncludeSubdomains {
		crawler.PageCollector.AllowedDomains = []string{}

		crawler.PageCollector.URLFilters = []*regexp.Regexp{
			extractor.New(
				extractor.WithHostPattern(`(?:(?:\w+[.])*` + regexp.QuoteMeta(crawler.cfg.Domain) + extractor.ExtractorPortOptionalPattern + `)`),
			).CompileRegex(),
		}
	}

	crawler.PageCollector.SetRequestTimeout(time.Duration(cfg.Timeout) * time.Second)

	if err = crawler.PageCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: cfg.Concurrency,
		RandomDelay: time.Duration(cfg.MaxRandomDelay) * time.Second,
	}); err != nil {
		return
	}

	if crawler.cfg.Debug {
		crawler.PageCollector.SetDebugger(&debug.LogDebugger{})
	}

	if crawler.cfg.Headers != nil && len(crawler.cfg.Headers) > 0 {
		crawler.PageCollector.OnRequest(func(request *colly.Request) {
			for index := range crawler.cfg.Headers {
				entry := crawler.cfg.Headers[index]

				var splitEntry []string

				if strings.Contains(entry, ": ") {
					splitEntry = strings.SplitN(entry, ": ", 2)
				} else if strings.Contains(entry, ":") {
					splitEntry = strings.SplitN(entry, ":", 2)
				} else {
					continue
				}

				header := strings.TrimSpace(splitEntry[0])
				value := splitEntry[1]

				request.Headers.Set(header, value)
			}
		})
	}

	extensions.Referer(crawler.PageCollector)

	if crawler.cfg.UserAgent == "" {
		crawler.PageCollector.UserAgent = DefaultUserAgent
	} else {
		switch ua := strings.ToLower(crawler.cfg.UserAgent); {
		case strings.HasPrefix(ua, "mob"):
			extensions.RandomMobileUserAgent(crawler.PageCollector)
		case strings.HasPrefix(ua, "web"):
			extensions.RandomUserAgent(crawler.PageCollector)
		default:
			crawler.PageCollector.UserAgent = crawler.cfg.UserAgent
		}
	}

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
		CheckRedirect: func(req *http.Request, via []*http.Request) (err error) {
			nextLocation := req.Response.Header.Get("Location")

			var parsedLocation *parser.URL

			parsedLocation, err = up.Parse(nextLocation)
			if err != nil {
				return
			}

			if parsedLocation.Domain == nil {
				return
			}

			if cfg.IncludeSubdomains && (parsedLocation.Domain.String() == cfg.Domain || strings.HasSuffix(parsedLocation.Domain.String(), "."+cfg.Domain)) {
				return
			}

			if parsedLocation.Domain.String() == cfg.Domain || parsedLocation.Domain.String() == "www."+cfg.Domain {
				return
			}

			return http.ErrUseLastResponse
		},
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

	crawler.FileCollector = crawler.PageCollector.Clone()
	crawler.FileCollector.URLFilters = nil

	crawler.PageCollector.ID = 1
	crawler.FileCollector.ID = 2

	return
}
