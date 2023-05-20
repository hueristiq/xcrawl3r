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
	hqurl "github.com/hueristiq/hqgoutils/url"
)

type Options struct { //nolint:govet // To be refactored
	Domain            string
	IncludeSubdomains bool
	Seeds             []string
	Concurrency       int
	Parallelism       int
	Debug             bool
	Delay             int
	Depth             int
	Headers           []string
	MaxRandomDelay    int // seconds
	Proxies           []string
	RenderTimeout     int // seconds
	Timeout           int // seconds
	UserAgent         string
}

type Crawler struct { //nolint:govet // To be refactored
	Domain                string
	IncludeSubdomains     bool
	Seeds                 []string
	Concurrency           int
	Parallelism           int
	Depth                 int
	Timeout               int
	MaxRandomDelay        int
	Debug                 bool
	Headers               []string
	UserAgent             string
	Proxies               []string
	PageCollector         *colly.Collector
	FileURLsRegex         *regexp.Regexp
	FileCollector         *colly.Collector
	URLsNotToRequestRegex *regexp.Regexp
	URLsRegex             *regexp.Regexp
}

func New(options *Options) (crawler *Crawler, err error) {
	crawler = &Crawler{
		Domain:            options.Domain,
		IncludeSubdomains: options.IncludeSubdomains,
		Seeds:             options.Seeds,
		Concurrency:       options.Concurrency,
		Parallelism:       options.Parallelism,
		Depth:             options.Depth,
		Timeout:           options.Timeout,
		MaxRandomDelay:    options.MaxRandomDelay,
		Debug:             options.Debug,
		Headers:           options.Headers,
		UserAgent:         options.UserAgent,
		Proxies:           options.Proxies,
	}

	crawler.URLsRegex = regexp.MustCompile(`(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\?|#][^"|']{0,}|)))(?:"|')`) //nolint:gocritic // Works fine!

	crawler.FileURLsRegex = regexp.MustCompile(`(?m).*?\.*(js|json|xml|csv|txt|map)(\?.*?|)$`) //nolint:gocritic // Works fine!

	crawler.URLsNotToRequestRegex = regexp.MustCompile(`(?i)\.(apng|bpm|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf|css)(?:\?|#|$)`)

	crawler.PageCollector = colly.NewCollector(
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.MaxDepth(crawler.Depth),
		colly.AllowedDomains(crawler.Domain, "www."+crawler.Domain),
	)

	if crawler.IncludeSubdomains {
		crawler.PageCollector.AllowedDomains = nil

		escapedDomain := regexp.QuoteMeta(crawler.Domain)
		pattern := fmt.Sprintf(`https?://([a-z0-9.-]*\.)?%s(/[a-zA-Z0-9()/*\-+_~:,.?#=]*)?`, escapedDomain)

		crawler.PageCollector.URLFilters = []*regexp.Regexp{
			regexp.MustCompile(pattern),
		}
	}

	crawler.PageCollector.SetRequestTimeout(time.Duration(crawler.Timeout) * time.Second)

	if err = crawler.PageCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: crawler.Concurrency,
		RandomDelay: time.Duration(crawler.MaxRandomDelay) * time.Second,
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

				if strings.Contains(entry, ": ") { //nolint:gocritic // Works!
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

	switch ua := strings.ToLower(crawler.UserAgent); {
	case strings.HasPrefix(ua, "mob"):
		extensions.RandomMobileUserAgent(crawler.PageCollector)
	case strings.HasPrefix(ua, "web"):
		extensions.RandomUserAgent(crawler.PageCollector)
	default:
		crawler.PageCollector.UserAgent = crawler.UserAgent
	}

	HTTPTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(crawler.Timeout) * time.Second,
			KeepAlive: time.Duration(crawler.Timeout) * time.Second,
		}).DialContext,
		MaxIdleConns:        100, // Golang default is 100
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     time.Duration(crawler.Timeout) * time.Second,
		TLSHandshakeTimeout: time.Duration(crawler.Timeout) * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // Intended
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
	}

	HTTPClient := &http.Client{
		Transport: HTTPTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) (err error) {
			nextLocation := req.Response.Header.Get("Location")

			var parsedLocation *hqurl.URL

			parsedLocation, err = hqurl.Parse(nextLocation)
			if err != nil {
				return err
			}

			if crawler.IncludeSubdomains &&
				(parsedLocation.Domain == crawler.Domain ||
					strings.HasSuffix(parsedLocation.Domain, "."+crawler.Domain)) {
				return nil
			}

			if parsedLocation.Domain == crawler.Domain || parsedLocation.Domain == "www."+crawler.Domain {
				return nil
			}

			return http.ErrUseLastResponse
		},
	}

	// NOTE: Must come BEFORE .SetClient calls
	crawler.PageCollector.SetClient(HTTPClient)

	// Proxies
	// NOTE: Must come AFTER .SetClient calls
	if len(crawler.Proxies) > 0 {
		var (
			rrps colly.ProxyFunc
		)

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

func (crawler *Crawler) Crawl() (URLsChannel chan URL) {
	URLsChannel = make(chan URL)

	go func() {
		defer close(URLsChannel)

		seedsChannel := make(chan string, crawler.Parallelism)

		go func() {
			defer close(seedsChannel)

			for index := range crawler.Seeds {
				seed := crawler.Seeds[index]

				seedsChannel <- seed
			}
		}()

		URLsWG := new(sync.WaitGroup)

		for i := 0; i < crawler.Parallelism; i++ {
			URLsWG.Add(1)

			go func() {
				defer URLsWG.Done()

				for seed := range seedsChannel {
					parsedSeed, err := hqurl.Parse(seed)
					if err != nil {
						continue
					}

					wg := &sync.WaitGroup{}
					seen := &sync.Map{}

					wg.Add(1)

					go func() {
						defer wg.Done()

						for URL := range crawler.sitemapParsing(parsedSeed) {
							_, loaded := seen.LoadOrStore(URL.Value, struct{}{})
							if loaded {
								continue
							}

							URLsChannel <- URL
						}
					}()

					wg.Add(1)

					go func() {
						defer wg.Done()

						for URL := range crawler.robotsParsing(parsedSeed) {
							_, loaded := seen.LoadOrStore(URL, struct{}{})
							if loaded {
								continue
							}

							URLsChannel <- URL
						}
					}()

					wg.Add(1)

					go func() {
						defer wg.Done()

						for URL := range crawler.pageCrawl(parsedSeed) {
							_, loaded := seen.LoadOrStore(URL, struct{}{})
							if loaded {
								continue
							}

							URLsChannel <- URL
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
