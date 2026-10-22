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
		includeSubdomains: c.cfg.IncludeSubdomains,
		maxPages:          c.cfg.MaxPages,
	}

	if len(c.cfg.Domains) > 0 {
		s.domains = make([]string, 0, len(c.cfg.Domains))

		for _, domain := range c.cfg.Domains {
			s.domains = append(s.domains, toASCIIHost(domain))
		}
	} else {
		s.domains = []string{toASCIIHost(parsedTargetURL.Hostname())}
	}

	return
}

func (s *scope) allows(rawURL string) (allowed bool) {
	parsedURL, err := up.Parse(rawURL)
	if err != nil {
		return
	}

	switch parsedURL.Scheme {
	case schemeHTTP, schemeHTTPS:
	default:
		return
	}

	host := toASCIIHost(parsedURL.Hostname())

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

func (s *scope) registrableAllows(rawURL string) (allowed bool) {
	parsedURL, err := up.Parse(rawURL)
	if err != nil {
		return
	}

	registrable, err := publicsuffix.EffectiveTLDPlusOne(toASCIIHost(parsedURL.Hostname()))
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

func (s *scope) widen(host string) (widened bool) {
	host = toASCIIHost(host)

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

// chargePage records one crawled HTML page against the page budget. It reports
// false once the budget is exhausted.
func (s *scope) chargePage() (allowed bool) {
	if s.maxPages <= 0 {
		return true
	}

	return s.requestCount.Add(1) <= uint32(s.maxPages) //nolint:gosec // maxPages validated >= 0 in New(); the > 0 guard here covers the rest.
}
