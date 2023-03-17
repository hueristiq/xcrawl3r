package hqcrawl3r

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/extensions"
	hqurl "github.com/hueristiq/hqgoutils/url"
)

type Crawler struct {
	URL                   *hqurl.URL
	Options               *Options
	PageCollector         *colly.Collector
	FileCollector         *colly.Collector
	URLsToLinkFindRegex   *regexp.Regexp
	URLsNotToRequestRegex *regexp.Regexp
}

type Options struct {
	TargetURL         *hqurl.URL
	Concurrency       int
	Debug             bool
	Delay             int
	Depth             int
	Headers           []string
	IncludeSubdomains bool
	MaxRandomDelay    int // seconds
	Proxy             string
	RenderTimeout     int // seconds
	Timeout           int // seconds
	UserAgent         string
}

var foundURLs sync.Map
var visitedURLs sync.Map

func New(options *Options) (crawler Crawler, err error) {
	crawler.URL = options.TargetURL
	crawler.Options = options

	crawler.PageCollector = colly.NewCollector(
		colly.AllowedDomains(crawler.URL.Domain, "www."+crawler.URL.Domain),
		colly.MaxDepth(crawler.Options.Depth),
		colly.IgnoreRobotsTxt(),
		colly.Async(true),
		colly.AllowURLRevisit(),
	)

	if crawler.Options.IncludeSubdomains {
		crawler.PageCollector.AllowedDomains = nil
		crawler.PageCollector.URLFilters = []*regexp.Regexp{
			regexp.MustCompile(".*(\\.|\\/\\/)" + strings.ReplaceAll(crawler.URL.Domain, ".", "\\.") + "((#|\\/|\\?).*)?"),
		}
	}

	if crawler.Options.Debug {
		crawler.PageCollector.SetDebugger(&debug.LogDebugger{})
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(crawler.Options.Timeout) * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:    100,
		MaxConnsPerHost: 1000,
		IdleConnTimeout: 30 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
	}

	if crawler.Options.Proxy != "" {
		var pU *url.URL

		pU, err = url.Parse(crawler.Options.Proxy)
		if err != nil {
			return
		}

		transport.Proxy = http.ProxyURL(pU)
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(crawler.Options.Timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			nextLocation := req.Response.Header.Get("Location")

			if strings.Contains(nextLocation, crawler.URL.Hostname()) {
				return nil
			}

			return http.ErrUseLastResponse
		},
	}

	crawler.PageCollector.SetClient(client)

	switch ua := strings.ToLower(crawler.Options.UserAgent); {
	case strings.HasPrefix(ua, "mobi"):
		extensions.RandomMobileUserAgent(crawler.PageCollector)
	case strings.HasPrefix(ua, "web"):
		extensions.RandomUserAgent(crawler.PageCollector)
	default:
		crawler.PageCollector.UserAgent = crawler.Options.UserAgent
	}

	if crawler.Options.Headers != nil && len(crawler.Options.Headers) > 0 {
		crawler.PageCollector.OnRequest(func(request *colly.Request) {
			for index := range crawler.Options.Headers {
				entry := crawler.Options.Headers[index]

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

	if err = crawler.PageCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: crawler.Options.Concurrency,
		Delay:       time.Duration(crawler.Options.Delay) * time.Second,
		RandomDelay: time.Duration(crawler.Options.MaxRandomDelay) * time.Second,
	}); err != nil {
		return
	}

	crawler.FileCollector = crawler.PageCollector.Clone()
	crawler.FileCollector.URLFilters = nil

	crawler.PageCollector.ID = 1
	crawler.FileCollector.ID = 2

	crawler.URLsToLinkFindRegex = regexp.MustCompile(`(?m).*?\.*(js|json|xml|csv|txt|map)(\?.*?|)$`)
	crawler.URLsNotToRequestRegex = regexp.MustCompile(`(?i)\.(apng|bpm|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf|css)(?:\?|#|$)`)

	return
}

func (crawler *Crawler) Crawl() (results chan string, err error) {
	crawler.PageCollector.OnRequest(func(request *colly.Request) {
		URL := strings.TrimRight(request.URL.String(), "/")

		if _, exists := visitedURLs.Load(URL); exists {
			request.Abort()

			return
		}

		if match := crawler.URLsNotToRequestRegex.MatchString(URL); match {
			request.Abort()

			return
		}

		if match := crawler.URLsToLinkFindRegex.MatchString(URL); match {
			if err = crawler.FileCollector.Visit(URL); err != nil {
				fmt.Println(err)
			}

			request.Abort()

			return
		}

		visitedURLs.Store(URL, struct{}{})

		return
	})

	crawler.FileCollector.OnResponse(func(response *colly.Response) {
		URL := strings.TrimRight(response.Request.URL.String(), "/")

		if _, exists := foundURLs.Load(URL); !exists {
			return
		}

		if err := crawler.record(URL); err != nil {
			return
		}

		foundURLs.Store(URL, struct{}{})
	})

	crawler.PageCollector.OnHTML("[href]", func(e *colly.HTMLElement) {
		relativeURL := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(relativeURL)

		if _, exists := foundURLs.Load(absoluteURL); exists {
			return
		}

		if err := crawler.record(absoluteURL); err != nil {
			return
		}

		foundURLs.Store(absoluteURL, struct{}{})

		if _, exists := visitedURLs.Load(absoluteURL); !exists {
			if err = e.Request.Visit(relativeURL); err != nil {
				return
			}
		}
	})

	crawler.PageCollector.OnHTML("[src]", func(e *colly.HTMLElement) {
		relativeURL := e.Attr("src")
		absoluteURL := e.Request.AbsoluteURL(relativeURL)

		if _, exists := foundURLs.Load(absoluteURL); exists {
			return
		}

		if err := crawler.record(absoluteURL); err != nil {
			return
		}

		foundURLs.Store(absoluteURL, struct{}{})

		if _, exists := visitedURLs.Load(absoluteURL); !exists {
			if err = e.Request.Visit(relativeURL); err != nil {
				return
			}
		}
	})

	crawler.FileCollector.OnRequest(func(request *colly.Request) {
		URL := request.URL.String()

		if _, exists := visitedURLs.Load(URL); exists {
			request.Abort()

			return
		}

		// If the URL is a `.min.js` (Minified JavaScript) try finding `.js`
		if strings.Contains(URL, ".min.js") {
			js := strings.ReplaceAll(URL, ".min.js", ".js")

			if _, exists := visitedURLs.Load(js); !exists {
				if err = crawler.FileCollector.Visit(js); err != nil {
					return
				}

				visitedURLs.Store(js, struct{}{})
			}
		}

		visitedURLs.Store(URL, struct{}{})
	})

	crawler.FileCollector.OnResponse(func(response *colly.Response) {
		links, err := crawler.FindLinks(string(response.Body))
		if err != nil {
			return
		}

		if len(links) < 1 {
			return
		}

		for _, link := range links {
			// Skip blank entries
			if len(link) <= 0 {
				continue
			}

			// Remove the single and double quotes from the parsed link on the ends
			link = strings.Trim(link, "\"")
			link = strings.Trim(link, "'")

			// Get the absolute URL
			absoluteURL := response.Request.AbsoluteURL(link)

			// Trim the trailing slash
			absoluteURL = strings.TrimRight(absoluteURL, "/")

			// Trim the spaces on either end (if any)
			absoluteURL = strings.Trim(absoluteURL, " ")
			if absoluteURL == "" {
				return
			}

			URL := crawler.fixURL(absoluteURL)

			if _, exists := foundURLs.Load(URL); !exists {
				if err := crawler.record(URL); err != nil {
					return
				}

				foundURLs.Store(URL, struct{}{})
			}

			if _, exists := visitedURLs.Load(URL); !exists {
				if err = crawler.PageCollector.Visit(URL); err != nil {
					return
				}
			}
		}
	})

	if err = crawler.PageCollector.Visit(crawler.URL.String()); err != nil {
		return
	}

	// Async means we must .Wait() on each Collector
	crawler.PageCollector.Wait()
	crawler.FileCollector.Wait()

	return
}
