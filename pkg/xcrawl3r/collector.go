package xcrawl3r

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
)

func (c *Crawler) collector(scope *scope) (collector *colly.Collector, err error) {
	collector = colly.NewCollector(
		colly.Async(true),
		colly.IgnoreRobotsTxt(),
		colly.MaxDepth(c.cfg.Depth),
	)

	if err = collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: c.cfg.Parallelism,
		Delay:       time.Duration(c.cfg.Delay) * time.Second,
		RandomDelay: time.Duration(c.cfg.RandomDelay) * time.Second,
	}); err != nil {
		return
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

	timeout := time.Duration(c.cfg.Timeout) * time.Second

	HTTPTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: timeout,
		}).DialContext,
		MaxIdleConns:        100,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     timeout,
		TLSHandshakeTimeout: timeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			Renegotiation:      tls.RenegotiateOnceAsClient,
		},
	}

	HTTPClient := &http.Client{
		Transport: &roundTripper{
			base:              HTTPTransport,
			retries:           c.cfg.Retries,
			requestsPerMinute: c.cfg.RequestsPerMinute,
		},
		Timeout: timeout,
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
	if len(c.cfg.Proxies) > 0 {
		var rrps colly.ProxyFunc

		rrps, err = proxy.RoundRobinProxySwitcher(c.cfg.Proxies...)
		if err != nil {
			return
		}

		collector.SetProxyFunc(rrps)
	}

	if c.cfg.Debug {
		collector.SetDebugger(&debug.LogDebugger{})
	}

	collector.SetStorage(c.storage)

	return
}
