package xcrawl3r

import (
	"strings"
)

type robotsRules struct {
	paths    []string
	sitemaps []string
}

func parseRobots(body []byte) (rules robotsRules) {
	content := strings.TrimPrefix(string(body), "\xef\xbb\xbf")

	for line := range strings.Lines(content) {
		if index := strings.IndexByte(line, '#'); index >= 0 {
			line = line[:index]
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		value = strings.TrimSpace(value)

		switch strings.ToLower(strings.TrimSpace(key)) {
		case "disallow", "allow":
			if value == "" || value == "/" {
				continue
			}

			if index := strings.IndexAny(value, "*$"); index >= 0 {
				value = value[:index]
			}

			if value == "" {
				continue
			}

			rules.paths = append(rules.paths, value)
		case "sitemap":
			if value == "" {
				continue
			}

			rules.sitemaps = append(rules.sitemaps, value)
		}
	}

	return
}
