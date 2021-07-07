package sigrawl3r

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

	"github.com/enenumxela/urlx/pkg/urlx"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/signedsecurity/sigrawl3r/pkg/sigrawl3r/exts"
)

type sigrawl3r struct {
	Options    *Options
	URL        *urlx.URL
	PCollector *colly.Collector
	JCollector *colly.Collector
}

func New(URL string, options *Options) (crawler sigrawl3r, err error) {
	crawler.Options = options

	parsedURL, err := urlx.Parse(URL)
	if err != nil {
		return crawler, err
	}

	crawler.URL = parsedURL

	eTLDPlus1 := parsedURL.ETLDPlus1
	escapedETLDPlus1 := strings.ReplaceAll(eTLDPlus1, ".", "\\.")

	// Instantiate default collector
	pCollector := colly.NewCollector(
		colly.Async(true),
		colly.MaxDepth(options.Depth),
	)

	if options.IncludeSubs {
		pCollector.URLFilters = []*regexp.Regexp{
			regexp.MustCompile(
				fmt.Sprintf(`(https?)://[^\s?#\/]*%s/?[^\s]*`, escapedETLDPlus1),
			),
		}
	} else {
		pCollector.AllowedDomains = []string{
			eTLDPlus1,
			"www." + eTLDPlus1,
		}
	}

	// Set User-Agent
	if options.UserAgent != "" {
		pCollector.UserAgent = options.UserAgent
	} else {
		extensions.RandomMobileUserAgent(pCollector)
	}

	// Referer
	extensions.Referer(pCollector)

	// Debug
	if options.Debug {
		pCollector.SetDebugger(&debug.LogDebugger{})
	}

	// Setup the client with our transport to pass to the collectors
	// NOTE: Must come BEFORE .SetClient calls
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(crawler.Options.Timeout) * time.Second,
			KeepAlive: time.Duration(crawler.Options.Timeout) * time.Second,
		}).DialContext,
		MaxIdleConns:    100, // Golang default is 100
		IdleConnTimeout: time.Duration(crawler.Options.Timeout) * time.Second,
	}

	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	client := &http.Client{
		Transport: tr,
	}

	pCollector.SetClient(client)

	// Setup proxy if supplied
	// NOTE: Must come AFTER .SetClient calls
	if crawler.Options.Proxies != "" {
		proxiesURLs := strings.Split(crawler.Options.Proxies, ",")

		rrps, err := proxy.RoundRobinProxySwitcher(proxiesURLs...)
		if err != nil {
			return crawler, err
		}

		pCollector.SetProxyFunc(rrps)
	}

	pCollector.SetRequestTimeout(
		time.Duration(crawler.Options.Timeout) * time.Second,
	)

	// Limit the number of threads started by colly to `crawler.Options.Threads`
	// when visiting links which domains' matches `*parsedURL.ETLDPlus1` glob
	err = pCollector.Limit(&colly.LimitRule{
		DomainGlob:  fmt.Sprintf("*%s", parsedURL.ETLDPlus1),
		Parallelism: crawler.Options.Threads,
		RandomDelay: time.Duration(crawler.Options.RandomDelay) * time.Second,
	})
	if err != nil {
		return crawler, err
	}

	jCollector := pCollector.Clone()
	jCollector.URLFilters = nil

	crawler.PCollector = pCollector
	crawler.JCollector = jCollector

	return crawler, nil
}

// Run is a
func (crawler *sigrawl3r) Run(URL string) (results Results, err error) {
	var URLs sync.Map
	var buckets sync.Map

	URLsSlice := make([]string, 0)
	bucketsSlice := make([]string, 0)

	jsRegex := regexp.MustCompile(`(?m).*?\.*(js|json|xml|csv|txt)(\?.*?|)$`)
	ignoreRegex := regexp.MustCompile(`(?m).*?\.*(jpg|png|gif|webp|psd|raw|bmp|heif|ico|css|pdf|jpeg|css|tif|tiff|ttf|woff|woff2|pdf|doc|svg|mp3|mp4|eot)(\?.*?|)$`)

	crawler.PCollector.OnRequest(func(request *colly.Request) {
		reqURL := request.URL.String()

		// If it's a javascript, json, xml, csv or txt file, ensure we pass it to the JCollector
		if match := jsRegex.MatchString(reqURL); match {
			// Minified JavaScript
			if strings.Contains(reqURL, ".min.js") {
				js := strings.ReplaceAll(reqURL, ".min.js", ".js")

				if _, exists := URLs.Load(js); exists {
					return
				}

				crawler.JCollector.Visit(js)

				URLs.Store(js, struct{}{})
			}

			crawler.JCollector.Visit(reqURL)

			// Cancel the request to ensure we don't process it on this collector
			request.Abort()
			return
		}

		// Is it an image or similar? Don't request it.
		if match := ignoreRegex.MatchString(reqURL); match {
			request.Abort()
			return
		}
	})

	crawler.PCollector.OnError(func(response *colly.Response, err error) {
		if response.StatusCode == 404 || response.StatusCode == 429 || response.StatusCode < 100 || response.StatusCode >= 500 {
			return
		}

		u := response.Request.URL.String()

		if _, exists := URLs.Load(u); exists {
			return
		}

		if ok := crawler.record("[url]", u); ok {
			URLsSlice = append(URLsSlice, u)
		}

		URLs.Store(u, struct{}{})
	})

	crawler.PCollector.OnResponse(func(response *colly.Response) {
		// s3 buckets
		S3s, err := exts.S3finder(string(response.Body))
		if err != nil {
			return
		}

		for _, S3 := range S3s {
			if _, exists := buckets.Load(S3); exists {
				return
			}

			fmt.Println("[s3]", S3)

			bucketsSlice = append(bucketsSlice, S3)
			buckets.Store(S3, struct{}{})
		}
	})

	crawler.PCollector.OnHTML("[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		// Get the absolute URL
		// An absolute URL provides all available data about a page’s location on the web.
		// Example: https://www.somewebsite.com/catalog/category/product
		absoluteURL := e.Request.AbsoluteURL(link)

		// Trim the trailing slash
		absoluteURL = strings.TrimRight(absoluteURL, "/")

		// Trim the spaces on either end (if any)
		absoluteURL = strings.Trim(absoluteURL, " ")

		if absoluteURL == "" {
			return
		}

		u, _ := url.Parse(absoluteURL)

		if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
			return
		}

		URL := fixURL(absoluteURL, crawler.URL)

		if _, exists := URLs.Load(URL); exists {
			return
		}

		e.Request.Visit(URL)

		if ok := crawler.record("[url]", URL); ok {
			URLsSlice = append(URLsSlice, URL)
		}

		URLs.Store(URL, struct{}{})
	})

	crawler.PCollector.OnHTML("script[src]", func(e *colly.HTMLElement) {
		link := e.Attr("src")

		// Get the absolute URL
		// An absolute URL provides all available data about a page’s location on the web.
		// Example: https://www.somewebsite.com/catalog/category/product
		absoluteURL := e.Request.AbsoluteURL(link)

		// Trim the trailing slash
		absoluteURL = strings.TrimRight(absoluteURL, "/")

		// Trim the spaces on either end (if any)
		absoluteURL = strings.Trim(absoluteURL, " ")
		if absoluteURL == "" {
			return
		}

		URL := fixURL(absoluteURL, crawler.URL)

		if _, exists := URLs.Load(URL); exists {
			return
		}

		crawler.PCollector.Visit(URL)

		if ok := crawler.record("[js]", URL); ok {
			URLsSlice = append(URLsSlice, URL)
		}

		URLs.Store(URL, struct{}{})
	})

	crawler.JCollector.OnResponse(func(response *colly.Response) {
		endpoints, err := exts.Linkfinder(string(response.Body))
		if err != nil {
			return
		}

		if len(endpoints) < 1 {
			return
		}

		for _, endpoint := range endpoints {
			// Skip blank entries
			if len(endpoint) <= 0 {
				continue
			}

			// Remove the single and double quotes from the parsed link on the ends
			endpoint = strings.Trim(endpoint, "\"")
			endpoint = strings.Trim(endpoint, "'")

			// Get the absolute URL
			absoluteURL := response.Request.AbsoluteURL(endpoint)

			// Trim the trailing slash
			absoluteURL = strings.TrimRight(absoluteURL, "/")

			// Trim the spaces on either end (if any)
			absoluteURL = strings.Trim(absoluteURL, " ")
			if absoluteURL == "" {
				return
			}

			URL := fixURL(absoluteURL, crawler.URL)

			if _, exists := URLs.Load(URL); exists {
				return
			}

			crawler.PCollector.Visit(URL)

			if ok := crawler.record("[linkfinder]", URL); ok {
				URLsSlice = append(URLsSlice, URL)
			}

			URLs.Store(URL, struct{}{})
		}

		// s3 buckets
		S3s, err := exts.S3finder(string(response.Body))
		if err != nil {
			return
		}

		for _, S3 := range S3s {
			if _, exists := buckets.Load(S3); exists {
				return
			}

			fmt.Println("[s3]", S3)

			bucketsSlice = append(bucketsSlice, S3)
			buckets.Store(S3, struct{}{})
		}
	})

	// setup a waitgroup to run all methods at the same time
	var wg sync.WaitGroup

	// colly
	wg.Add(1)
	go func() {
		defer wg.Done()

		crawler.PCollector.Visit(crawler.URL.String())
	}()

	wg.Wait()
	crawler.PCollector.Wait()
	crawler.JCollector.Wait()

	results.URLs = URLsSlice
	results.Buckets = bucketsSlice

	return results, nil
}

func (crawler *sigrawl3r) record(tag string, URL string) (print bool) {
	URL = decode(URL)

	parsedURL, err := urlx.Parse(URL)
	if err != nil {
		return false
	}

	if crawler.Options.IncludeSubs {
		escapedHost := strings.ReplaceAll(crawler.URL.Host, ".", "\\.")
		print, _ = regexp.MatchString(".*(\\.|\\/\\/)"+escapedHost+"((#|\\/|\\?).*)?", URL)
	} else {
		print = parsedURL.Host == crawler.URL.Host || parsedURL.Host == "www."+crawler.URL.Host
	}

	if print {
		fmt.Println(tag, URL)
	}

	return print
}
