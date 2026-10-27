package xcrawl3r

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	hqgohttpheader "github.com/hueristiq/hq-lib-http-go/header"
	hqgohttpuseragent "github.com/hueristiq/hq-lib-http-go/header/useragent"
)

func (c *Crawler) collector(ctx context.Context, scope *scope, results chan<- Result) (collector *colly.Collector, err error) {
	collector = colly.NewCollector(
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.MaxDepth(c.cfg.Depth),
		colly.StdlibContext(ctx),
	)

	if err = collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: c.cfg.Parallelism,
		Delay:       c.cfg.Delay,
		RandomDelay: c.cfg.RandomDelay,
	}); err != nil {
		return
	}

	if c.cfg.RandomUserAgent {
		collector.OnRequest(func(request *colly.Request) {
			request.Headers.Set(hqgohttpheader.UserAgent.String(), hqgohttpuseragent.Random())
		})
	} else if c.cfg.Client != nil && c.cfg.Client.UserAgent != "" {
		collector.OnRequest(func(request *colly.Request) {
			request.Headers.Set(hqgohttpheader.UserAgent.String(), c.cfg.Client.UserAgent)
		})
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

	httpTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   c.cfg.Timeout,
			KeepAlive: c.cfg.Timeout,
		}).DialContext,
		MaxIdleConns:        100,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     c.cfg.Timeout,
		TLSHandshakeTimeout: c.cfg.Timeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.cfg.InsecureSkipVerify, //nolint:gosec // user-controllable via Configuration.InsecureSkipVerify/--insecure-skip-verify, default insecure.
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
	}

	// Assign the proxy to the transport rather than calling
	// collector.SetProxyFunc: colly replaces the client's transport when the
	// proxy function is set, dropping retries, rate limiting, and TLS options.
	if len(c.cfg.Proxies) > 0 {
		var proxyFunc colly.ProxyFunc

		proxyFunc, err = proxy.RoundRobinProxySwitcher(c.cfg.Proxies...)
		if err != nil {
			return
		}

		httpTransport.Proxy = proxyFunc
	}

	httpClient := &http.Client{
		Transport: &roundTripper{
			base:              httpTransport,
			retries:           c.cfg.Retries,
			requestsPerMinute: c.cfg.RequestsPerMinute,
		},
		Timeout: c.cfg.Timeout,
		CheckRedirect: func(request *http.Request, _ []*http.Request) (err error) {
			return checkRedirect(ctx, results, scope, request)
		},
	}

	httpClient.Jar, err = cookiejar.New(nil)
	if err != nil {
		return
	}

	// NOTE: Must come BEFORE .SetClient calls
	collector.SetClient(httpClient)

	if err = collector.SetStorage(c.storage); err != nil {
		return
	}

	return
}

func checkRedirect(ctx context.Context, results chan<- Result, scope *scope, request *http.Request) (err error) {
	if scope.allows(request.URL.String()) {
		return
	}

	// Redirects to a sibling subdomain are followed only when subdomains are
	// explicitly in scope. Everything else keeps the last in-scope response.
	if !scope.includeSubdomains || !scope.registrableAllows(request.URL.String()) {
		return http.ErrUseLastResponse
	}

	host := toASCIIHost(request.URL.Hostname())

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
