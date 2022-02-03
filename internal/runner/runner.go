package runner

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
	"github.com/signedsecurity/sigrawl3r/internal/options"
	"github.com/signedsecurity/sigrawl3r/internal/runner/exts"
)

type Runner struct {
	URL        *urlx.URL
	Options    *options.Options
	PCollector *colly.Collector
	JCollector *colly.Collector
}

func New(URL string, options *options.Options) (runner Runner, err error) {
	runner.URL, err = urlx.Parse(URL)
	if err != nil {
		return runner, err
	}

	runner.Options = options

	eTLDPlus1 := runner.URL.ETLDPlus1
	escapedETLDPlus1 := strings.ReplaceAll(eTLDPlus1, ".", "\\.")

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
			Timeout:   time.Duration(runner.Options.Timeout) * time.Second,
			KeepAlive: time.Duration(runner.Options.Timeout) * time.Second,
		}).DialContext,
		MaxIdleConns:    100, // Golang default is 100
		IdleConnTimeout: time.Duration(runner.Options.Timeout) * time.Second,
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
	if runner.Options.Proxies != "" {
		proxiesURLs := strings.Split(runner.Options.Proxies, ",")

		rrps, err := proxy.RoundRobinProxySwitcher(proxiesURLs...)
		if err != nil {
			return runner, err
		}

		pCollector.SetProxyFunc(rrps)
	}

	pCollector.SetRequestTimeout(
		time.Duration(runner.Options.Timeout) * time.Second,
	)

	// Limit the number of threads started by colly to `runner.Options.Threads`
	// when visiting links which domains' matches `*parsedURL.ETLDPlus1` glob
	err = pCollector.Limit(&colly.LimitRule{
		DomainGlob:  fmt.Sprintf("*%s", runner.URL.ETLDPlus1),
		Parallelism: runner.Options.Threads,
		RandomDelay: time.Duration(runner.Options.RandomDelay) * time.Second,
	})
	if err != nil {
		return runner, err
	}

	jCollector := pCollector.Clone()
	jCollector.URLFilters = nil

	runner.PCollector = pCollector
	runner.JCollector = jCollector

	return runner, nil
}

// Run is a
func (runner *Runner) Run(URL string) (results Results, err error) {
	var URLs sync.Map

	URLsSlice := make([]string, 0)

	jsRegex := regexp.MustCompile(`(?m).*?\.*(js|json|xml|csv|txt)(\?.*?|)$`)
	ignoreRegex := regexp.MustCompile(`(?m).*?\.*(jpg|png|gif|webp|psd|raw|bmp|heif|ico|css|pdf|jpeg|css|tif|tiff|ttf|woff|woff2|pdf|doc|svg|mp3|mp4|eot)(\?.*?|)$`)

	runner.PCollector.OnRequest(func(request *colly.Request) {
		reqURL := request.URL.String()

		// If it's a javascript, json, xml, csv or txt file, ensure we pass it to the JCollector
		if match := jsRegex.MatchString(reqURL); match {
			// Minified JavaScript
			if strings.Contains(reqURL, ".min.js") {
				js := strings.ReplaceAll(reqURL, ".min.js", ".js")

				if _, exists := URLs.Load(js); exists {
					return
				}

				runner.JCollector.Visit(js)

				URLs.Store(js, struct{}{})
			}

			runner.JCollector.Visit(reqURL)

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

	runner.PCollector.OnError(func(response *colly.Response, err error) {
		if response.StatusCode == 404 || response.StatusCode == 429 || response.StatusCode < 100 || response.StatusCode >= 500 {
			return
		}

		u := response.Request.URL.String()

		if _, exists := URLs.Load(u); exists {
			return
		}

		if ok := runner.record("[url]", u); ok {
			URLsSlice = append(URLsSlice, u)
		}

		URLs.Store(u, struct{}{})
	})

	runner.PCollector.OnHTML("[href]", func(e *colly.HTMLElement) {
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

		URL := fixURL(absoluteURL, runner.URL)

		if _, exists := URLs.Load(URL); exists {
			return
		}

		e.Request.Visit(URL)

		if ok := runner.record("[url]", URL); ok {
			URLsSlice = append(URLsSlice, URL)
		}

		URLs.Store(URL, struct{}{})
	})

	runner.PCollector.OnHTML("script[src]", func(e *colly.HTMLElement) {
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

		URL := fixURL(absoluteURL, runner.URL)

		if _, exists := URLs.Load(URL); exists {
			return
		}

		runner.PCollector.Visit(URL)

		if ok := runner.record("[js]", URL); ok {
			URLsSlice = append(URLsSlice, URL)
		}

		URLs.Store(URL, struct{}{})
	})

	runner.JCollector.OnResponse(func(response *colly.Response) {
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

			URL := fixURL(absoluteURL, runner.URL)

			if _, exists := URLs.Load(URL); exists {
				return
			}

			runner.PCollector.Visit(URL)

			if ok := runner.record("[linkfinder]", URL); ok {
				URLsSlice = append(URLsSlice, URL)
			}

			URLs.Store(URL, struct{}{})
		}
	})

	// setup a waitgroup to run all methods at the same time
	var wg sync.WaitGroup

	// colly
	wg.Add(1)
	go func() {
		defer wg.Done()

		runner.PCollector.Visit(runner.URL.String())
	}()

	wg.Wait()
	runner.PCollector.Wait()
	runner.JCollector.Wait()

	results.URLs = URLsSlice

	return results, nil
}

func (crawler *Runner) record(tag string, URL string) (print bool) {
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
