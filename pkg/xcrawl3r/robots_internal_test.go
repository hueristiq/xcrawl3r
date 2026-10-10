package xcrawl3r

import (
	"slices"
	"testing"
)

func TestParseRobots(t *testing.T) {
	t.Parallel()

	const adminPath = "/admin"

	tests := []struct {
		name     string
		body     string
		paths    []string
		sitemaps []string
	}{
		{
			name:  "disallow and allow paths",
			body:  "User-agent: *\nDisallow: " + adminPath + "\nAllow: /admin/public\n",
			paths: []string{adminPath, "/admin/public"},
		},
		{
			name:  "case insensitive keys",
			body:  "DISALLOW: /private\ndisallow: /tmp\n",
			paths: []string{"/private", "/tmp"},
		},
		{
			name:  "comments stripped",
			body:  "Disallow: " + adminPath + " # admin panel\n# Disallow: /ignored\n",
			paths: []string{adminPath},
		},
		{
			name:  "wildcards cut",
			body:  "Disallow: /private/*\nDisallow: /*.pdf$\n",
			paths: []string{"/private/", "/"},
		},
		{
			name: "empty and root values skipped",
			body: "Disallow:\nDisallow: /\nAllow:\n",
		},
		{
			name:  "malformed lines skipped",
			body:  "garbage line\nDisallow: " + adminPath + "\n: orphan\n",
			paths: []string{adminPath},
		},
		{
			name:     "sitemaps collected",
			body:     "Sitemap: https://example.com/sitemap.xml\nSitemap: https://example.com/news.xml\n",
			sitemaps: []string{"https://example.com/sitemap.xml", "https://example.com/news.xml"},
		},
		{
			name:     "empty sitemap skipped",
			body:     "Sitemap:\n",
			sitemaps: nil,
		},
		{
			name:  "bom stripped",
			body:  "\xef\xbb\xbfDisallow: /admin\n",
			paths: []string{"/admin"},
		},
		{
			name:  "all groups harvested",
			body:  "User-agent: googlebot\nDisallow: /a\n\nUser-agent: bingbot\nDisallow: /b\n",
			paths: []string{"/a", "/b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rules := parseRobots([]byte(tt.body))

			if !slices.Equal(rules.paths, tt.paths) {
				t.Errorf("paths = %v, want %v", rules.paths, tt.paths)
			}

			if !slices.Equal(rules.sitemaps, tt.sitemaps) {
				t.Errorf("sitemaps = %v, want %v", rules.sitemaps, tt.sitemaps)
			}
		})
	}
}
