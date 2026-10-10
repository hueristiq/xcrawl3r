package xcrawl3r

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
)

func (c *Crawler) collector(ctx context.Context, scope *scope) (collector *colly.Collector, err error) {
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
			InsecureSkipVerify: true,
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
		CheckRedirect: func(req *http.Request, _ []*http.Request) (err error) {
			if !scope.allows(req.URL.String()) {
				return http.ErrUseLastResponse
			}

			return
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

	if c.opts.debug {
		collector.SetDebugger(&debug.LogDebugger{})
	}

	collector.SetStorage(c.storage)

	return
}
