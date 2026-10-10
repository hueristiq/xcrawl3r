package xcrawl3r

import (
	"strings"

	hqgourlparser "github.com/hueristiq/hq-lib-url-go/parser"
)

type scope struct {
	domains           []string
	includeSubdomains bool
}

func (c *Crawler) scopeFor(parsedTargetURL *hqgourlparser.URL) (s *scope) {
	s = &scope{
		includeSubdomains: c.cfg.IncludeSubdomains,
	}

	if len(c.cfg.Domains) > 0 {
		s.domains = make([]string, 0, len(c.cfg.Domains))

		for _, domain := range c.cfg.Domains {
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
