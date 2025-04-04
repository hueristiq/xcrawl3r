package xcrawl3r

import (
	"strings"
)

func (crawler *Crawler) IsInScope(URL string) (isInScope bool) {
	parsedURL, err := up.Parse(URL)
	if err != nil {
		return
	}

	if parsedURL.Domain == nil {
		return
	}

	if crawler.cfg.IncludeSubdomains {
		isInScope = parsedURL.Domain.String() == crawler.cfg.Domain || strings.HasSuffix(parsedURL.Domain.String(), "."+crawler.cfg.Domain)
	} else {
		isInScope = parsedURL.Domain.String() == crawler.cfg.Domain || parsedURL.Domain.String() == "www."+crawler.cfg.Domain
	}

	return
}
