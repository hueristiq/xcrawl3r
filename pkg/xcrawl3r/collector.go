package xcrawl3r

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	hqgohttpheader "github.com/hueristiq/hq-lib-http-go/header"
	hqgohttpuseragent "github.com/hueristiq/hq-lib-http-go/useragent"
)

func (c *Crawler) collector(ctx context.Context, scope *scope, results chan<- Result) (collector *colly.Collector, err error) {
	collector = colly.NewCollector(
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.MaxDepth(c.opts.depth),
		colly.StdlibContext(ctx),
	)

	if err = collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: c.opts.parallelism,
		Delay:       c.opts.delay,
		RandomDelay: c.opts.randomDelay,
	}); err != nil {
		return
	}

	if c.opts.randomUserAgent {
		collector.OnRequest(func(request *colly.Request) {
			request.Headers.Set(hqgohttpheader.UserAgent.String(), hqgohttpuseragent.Random())
		})
	}

	if len(c.opts.headers) > 0 {
		collector.OnRequest(func(request *colly.Request) {
			for _, entry := range c.opts.headers {
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
			Timeout:   c.opts.timeout,
			KeepAlive: c.opts.timeout,
		}).DialContext,
		MaxIdleConns:        100,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     c.opts.timeout,
		TLSHandshakeTimeout: c.opts.timeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.opts.insecureSkipVerify, //nolint:gosec // user-controllable via WithInsecureSkipVerify/--insecure-skip-verify, default insecure.
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
	}

	HTTPClient := &http.Client{
		Transport: &roundTripper{
			base:              HTTPTransport,
			retries:           c.opts.retries,
			requestsPerMinute: c.opts.requestsPerMinute,
		},
		Timeout: c.opts.timeout,
		CheckRedirect: func(request *http.Request, _ []*http.Request) (err error) {
			return checkRedirect(ctx, results, scope, request)
		},
	}

	// NOTE: Must come BEFORE .SetClient calls
	collector.SetClient(HTTPClient)

	// NOTE: Must come AFTER .SetClient calls
	if len(c.opts.proxies) > 0 {
		var rrps colly.ProxyFunc

		rrps, err = proxy.RoundRobinProxySwitcher(c.opts.proxies...)
		if err != nil {
			return
		}

		collector.SetProxyFunc(rrps)
	}

	if err = collector.SetStorage(c.storage); err != nil {
		return
	}

	return
}

func checkRedirect(ctx context.Context, results chan<- Result, scope *scope, request *http.Request) (err error) {
	if scope.allows(request.URL.String()) {
		return
	}

	if !scope.registrableAllows(request.URL.String()) {
		return http.ErrUseLastResponse
	}

	host := strings.ToLower(request.URL.Hostname())

	if scope.widen(host) {
		emit(ctx, results, Result{
			Type: ResultWarning,
			Value: fmt.Sprintf(
				"redirect to %s is outside the configured scope; following it and adding %s to the scope",
				request.URL.String(),
				host,
			),
		})
	}

	return
}
