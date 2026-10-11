package xcrawl3r

import (
	"strings"
	"sync"
	"sync/atomic"

	hqgourlparser "github.com/hueristiq/hq-lib-url-go/parser"
	"golang.org/x/net/publicsuffix"
)

type scope struct {
	domains           []string
	includeSubdomains bool

	maxPages     int
	requestCount atomic.Uint32

	mu sync.RWMutex
}

func (c *Crawler) scopeFor(parsedTargetURL *hqgourlparser.URL) (s *scope) {
	s = &scope{
		includeSubdomains: c.opts.includeSubdomains,
		maxPages:          c.opts.maxPages,
	}

	if len(c.opts.domains) > 0 {
		s.domains = make([]string, 0, len(c.opts.domains))

		for _, domain := range c.opts.domains {
			s.domains = append(s.domains, strings.ToLower(domain))
		}
	} else {
		s.domains = []string{strings.ToLower(parsedTargetURL.Hostname())}
	}

	return
}

func (s *scope) allows(rawURL string) (allowed bool) {
	parsedURL, err := up.Parse(rawURL)
	if err != nil {
		return
	}

	switch parsedURL.Scheme {
	case "http", "https":
	default:
		return
	}

	host := strings.ToLower(parsedURL.Hostname())

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, domain := range s.domains {
		if host == domain {
			return true
		}

		if s.includeSubdomains && strings.HasSuffix(host, "."+domain) {
			return true
		}
	}

	return
}

// registrableAllows reports whether rawURL's host shares a registrable domain
// (public suffix + one label, e.g. example.co.uk) with one of the scoped
// domains, even when the host itself is out of scope.
func (s *scope) registrableAllows(rawURL string) (allowed bool) {
	parsedURL, err := up.Parse(rawURL)
	if err != nil {
		return
	}

	registrable, err := publicsuffix.EffectiveTLDPlusOne(strings.ToLower(parsedURL.Hostname()))
	if err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, domain := range s.domains {
		domainRegistrable, err := publicsuffix.EffectiveTLDPlusOne(domain)
		if err != nil {
			continue
		}

		if registrable == domainRegistrable {
			return true
		}
	}

	return
}

// widen adds host to the scoped domains when it is not already covered,
// e.g. when a redirect moved the crawl from example.com to www.example.com.
func (s *scope) widen(host string) (widened bool) {
	host = strings.ToLower(host)

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, domain := range s.domains {
		if host == domain || (s.includeSubdomains && strings.HasSuffix(host, "."+domain)) {
			return
		}
	}

	s.domains = append(s.domains, host)

	return true
}
