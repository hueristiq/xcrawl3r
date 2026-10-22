//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"compress/gzip"
	"slices"
	"testing"
)

func TestParseRobots(t *testing.T) {
	t.Parallel()

	body := []byte("\xef\xbb\xbfUser-agent: *\n" +
		"Disallow: /private\n" +
		"Allow: /public # comment\n" +
		"Disallow: /\n" +
		"Disallow: /wild*card\n" +
		"Sitemap: https://example.com/sitemap.xml\n" +
		"Sitemap:\n")

	got := parseRobots(body)

	wantPaths := []string{"/private", "/public", "/wild"}
	if !slices.Equal(got.paths, wantPaths) {
		t.Fatalf("paths = %v, want %v", got.paths, wantPaths)
	}

	wantSitemaps := []string{"https://example.com/sitemap.xml"}
	if !slices.Equal(got.sitemaps, wantSitemaps) {
		t.Fatalf("sitemaps = %v, want %v", got.sitemaps, wantSitemaps)
	}
}

func TestIsSitemapPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawPath string
		want    bool
	}{
		{name: "sitemap xml", rawPath: "/sitemap.xml", want: true},
		{name: "sitemap gz", rawPath: "/sitemap.xml.gz", want: true},
		{name: "uppercase", rawPath: "/SITEMAP.XML", want: true},
		{name: "sitemap html", rawPath: "/sitemap.html", want: false},
		{name: "unrelated", rawPath: testIndexPath, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isSitemapPath(tt.rawPath); got != tt.want {
				t.Fatalf("isSitemapPath(%q) = %v, want %v", tt.rawPath, got, tt.want)
			}
		})
	}
}

func TestIsNonGzippedXMLGZProbe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rawURL string
		err    error
		want   bool
	}{
		{name: "gzip probe", rawURL: "https://example.com/sitemap.xml.gz", err: gzip.ErrHeader, want: true},
		{name: "uppercase gzip probe", rawURL: "https://example.com/SITEMAP.XML.GZ", err: gzip.ErrHeader, want: true},
		{name: "other url", rawURL: "https://example.com/index.html", err: gzip.ErrHeader, want: false},
		{name: "other error", rawURL: "https://example.com/sitemap.xml.gz", err: gzip.ErrChecksum, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isNonGzippedXMLGZProbe(tt.rawURL, tt.err); got != tt.want {
				t.Fatalf("isNonGzippedXMLGZProbe() = %v, want %v", got, tt.want)
			}
		})
	}
}
