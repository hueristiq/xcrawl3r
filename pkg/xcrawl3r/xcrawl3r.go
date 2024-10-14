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
	hqgourl "github.com/hueristiq/hq-go-url"
	"github.com/hueristiq/xcrawl3r/internal/configuration"
)

type Crawler struct {
	Domain            string
	IncludeSubdomains bool
	Seeds             []string

	Headless bool
	Headers  []string
	Proxies  []string
	Render   bool
	// Timeout   int
	UserAgent string

	// Concurrency    int
	Delay int
	// MaxRandomDelay int
	Parallelism int

	Debug bool

	FileURLsRegex *regexp.Regexp

	URLsNotToRequestRegex *regexp.Regexp
	URLsRegex             *regexp.Regexp

	PageCollector *colly.Collector
	FileCollector *colly.Collector
}

func (crawler *Crawler) Crawl() (results chan Result) {
	results = make(chan Result)

	go func() {
		defer close(results)

		seedsChannel := make(chan string, crawler.Parallelism)

		go func() {
			defer close(seedsChannel)

			for index := range crawler.Seeds {
				seed := crawler.Seeds[index]

				seedsChannel <- seed
			}
		}()

		URLsWG := new(sync.WaitGroup)

		for range crawler.Parallelism {
			URLsWG.Add(1)

			go func() {
				defer URLsWG.Done()

				for seed := range seedsChannel {
					parsedSeed, err := up.Parse(seed)
					if err != nil {
						continue
					}

					seenURLs := &sync.Map{}

					wg := &sync.WaitGroup{}

					wg.Add(1)

					go func() {
						defer wg.Done()

						for URL := range crawler.sitemapParsing(parsedSeed) {
							_, loaded := seenURLs.LoadOrStore(URL.Value, struct{}{})
							if loaded {
								continue
							}

							results <- URL
						}
					}()

					wg.Add(1)

					go func() {
						defer wg.Done()

						for URL := range crawler.robotsParsing(parsedSeed) {
							_, loaded := seenURLs.LoadOrStore(URL, struct{}{})
							if loaded {
								continue
							}

							results <- URL
						}
					}()

					wg.Add(1)

					go func() {
						defer wg.Done()

						for URL := range crawler.pageCrawl(parsedSeed) {
							_, loaded := seenURLs.LoadOrStore(URL, struct{}{})
							if loaded {
								continue
							}

							results <- URL
						}
					}()

					wg.Wait()
				}
			}()
		}

		URLsWG.Wait()
	}()

	return
}

type Configuration struct {
	Depth int

	Domain            string
	IncludeSubdomains bool
	Seeds             []string

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
}

var (
	DefaultUserAgent = fmt.Sprintf("%s v%s (https://github.com/hueristiq/%s)", configuration.NAME, configuration.VERSION, configuration.NAME)
	up               = hqgourl.NewParser()
)

func New(cfg *Configuration) (crawler *Crawler, err error) {
	crawler = &Crawler{
		Domain:            cfg.Domain,
		IncludeSubdomains: cfg.IncludeSubdomains,
		Seeds:             cfg.Seeds,

		Headless: cfg.Headless,
		Headers:  cfg.Headers,
		Proxies:  cfg.Proxies,
		Render:   cfg.Render,
		// Timeout:   cfg.Timeout,
		UserAgent: cfg.UserAgent,

		// Concurrency:    cfg.Concurrency,
		Delay: cfg.Delay,
		// MaxRandomDelay: cfg.MaxRandomDelay,
		Parallelism: cfg.Parallelism,

		Debug: cfg.Debug,
	}

	crawler.URLsRegex = hqgourl.NewExtractor().CompileRegex()

	crawler.FileURLsRegex = regexp.MustCompile(`(?m).*?\.*(js|json|xml|csv|txt|map)(\?.*?|)$`)

	crawler.URLsNotToRequestRegex = regexp.MustCompile(`(?i)\.(apng|bpm|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf|css)(?:\?|#|$)`)

	crawler.PageCollector = colly.NewCollector(
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.MaxDepth(cfg.Depth),
		colly.AllowedDomains(crawler.Domain, "www."+crawler.Domain),
	)

	if crawler.IncludeSubdomains {
		crawler.PageCollector.AllowedDomains = []string{}

		// pattern := fmt.Sprintf(`https?://([a-z0-9.-]*\.)?%s(/[a-zA-Z0-9()/*\-+_~:,.?#=]*)?`, regexp.QuoteMeta(crawler.Domain))

		crawler.PageCollector.URLFilters = []*regexp.Regexp{
			// regexp.MustCompile(pattern),
			hqgourl.NewExtractor(hqgourl.ExtractorWithSchemePattern(`(?:https?)://`)).CompileRegex(),
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

	if crawler.Debug {
		crawler.PageCollector.SetDebugger(&debug.LogDebugger{})
	}

	if crawler.Headers != nil && len(crawler.Headers) > 0 {
		crawler.PageCollector.OnRequest(func(request *colly.Request) {
			for index := range crawler.Headers {
				entry := crawler.Headers[index]

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

	if crawler.UserAgent == "" {
		crawler.PageCollector.UserAgent = DefaultUserAgent
	} else {
		switch ua := strings.ToLower(crawler.UserAgent); {
		case strings.HasPrefix(ua, "mob"):
			extensions.RandomMobileUserAgent(crawler.PageCollector)
		case strings.HasPrefix(ua, "web"):
			extensions.RandomUserAgent(crawler.PageCollector)
		default:
			crawler.PageCollector.UserAgent = crawler.UserAgent
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

			var parsedLocation *hqgourl.URL

			parsedLocation, err = up.Parse(nextLocation)
			if err != nil {
				return
			}

			if parsedLocation.Domain == nil {
				return
			}

			fmt.Println(parsedLocation)

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
	if len(crawler.Proxies) > 0 {
		var rrps colly.ProxyFunc

		rrps, err = proxy.RoundRobinProxySwitcher(crawler.Proxies...)
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
