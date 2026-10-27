//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"net/url"
	"testing"
)

func TestMaybePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		link string
		want bool
	}{
		{name: "root", link: "/", want: false},
		{name: "absolute path link", link: "/foo", want: true},
		{name: "path with query", link: "/foo?bar=1", want: true},
		{name: "known extension", link: "foo.js", want: true},
		{name: "known text extension", link: "foo.txt", want: true},
		{name: "unknown extension", link: "foo.unknown", want: false},
		{name: "bare word", link: "foo", want: false},
		{name: "absolute url link", link: testPathURL, want: true},
		{name: "non-http url", link: "mailto:user@example.com", want: false},
		{name: "query only", link: "?q=1", want: true},
		{name: "scheme relative", link: "//example.com/path", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := maybePath(tt.link); got != tt.want {
				t.Fatalf("maybePath(%q) = %v, want %v", tt.link, got, tt.want)
			}
		})
	}
}

func TestHandleJSLiteralsFrom(t *testing.T) {
	t.Parallel()

	crawler := newTestCrawler(t, &Configuration{Domains: []string{testDomain}})
	scope := &scope{domains: []string{testDomain}}

	base, err := url.Parse("https://example.com/app.js")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	body := `
const skipped = "/skipped.js";
fetch("/api/users");
const joined = "/api" + "/v1";
`

	var visited []string

	visit := func(rawURL string) error {
		visited = append(visited, rawURL)

		return nil
	}

	results := make(chan Result, 128)

	crawler.handleJSLiteralsFrom(t.Context(), base, visit, body, scope, results)

	close(results)

	var values []string

	for result := range results {
		values = append(values, result.Value)
	}

	assertContains(t, visited, "https://example.com/api/users")
	assertContains(t, visited, "https://example.com/api/v1")
	assertNotContains(t, visited, "https://example.com/skipped.js")
	assertContains(t, values, "https://example.com/api/users")
	assertContains(t, values, "https://example.com/api/v1")
}

func TestHandleJSSnippetFrom(t *testing.T) {
	t.Parallel()

	crawler := newTestCrawler(t, &Configuration{Domains: []string{testDomain}})
	scope := &scope{domains: []string{testDomain}}

	base, err := url.Parse("https://example.com/src/app.ts")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	var visited []string

	visit := func(rawURL string) error {
		visited = append(visited, rawURL)

		return nil
	}

	results := make(chan Result, 128)

	crawler.handleJSSnippetFrom(t.Context(), base, visit, "fetch('./relative.js')", "test", scope, results)

	close(results)

	var values []string

	for result := range results {
		values = append(values, result.Value)
	}

	assertContains(t, visited, "https://example.com/src/relative.js")
	assertContains(t, values, "https://example.com/src/relative.js")
}
