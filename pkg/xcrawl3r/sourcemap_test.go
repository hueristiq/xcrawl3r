//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"net/url"
	"testing"
)

func TestResolveSourceRoot(t *testing.T) {
	t.Parallel()

	mapURL, err := url.Parse("https://example.com/js/app.js.map")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	tests := []struct {
		name       string
		sourceRoot string
		want       string
	}{
		{name: "relative path", sourceRoot: "src", want: "https://example.com/js/src/"},
		{name: "relative path with slash", sourceRoot: "src/", want: "https://example.com/js/src/"},
		{name: "absolute root path", sourceRoot: "/src", want: testSourceRootURL},
		{name: "absolute root url", sourceRoot: "https://cdn.example.com/src", want: "https://cdn.example.com/src/"},
		{name: "empty root", sourceRoot: "", want: testRootURL},
		{name: "invalid root", sourceRoot: "%zz", want: "https://example.com/js/app.js.map"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := resolveSourceRoot(mapURL, tt.sourceRoot); got.String() != tt.want {
				t.Fatalf("resolveSourceRoot(%q) = %q, want %q", tt.sourceRoot, got, tt.want)
			}
		})
	}
}

func TestHandleSourceMap(t *testing.T) {
	t.Parallel()

	t.Run("sourceRoot and sourcesContent base", func(t *testing.T) {
		t.Parallel()

		crawler := newTestCrawler(t, &Configuration{
			Domains:           []string{testDomain},
			IncludeSubdomains: true,
		})

		scope := &scope{
			domains:           []string{testDomain},
			includeSubdomains: true,
		}

		mapURL, err := url.Parse("https://example.com/maps/app.js.map")
		if err != nil {
			t.Fatalf("url.Parse() error = %v", err)
		}

		body := []byte(`{"version":3,"sourceRoot":"../src","sources":["app.ts"],"sourcesContent":["fetch('./relative.js')"]}`)

		var visited []string

		visit := func(rawURL string) error {
			visited = append(visited, rawURL)

			return nil
		}

		results := make(chan Result, 128)

		crawler.handleSourceMap(t.Context(), mapURL, visit, body, scope, results)

		close(results)

		var values []string

		for result := range results {
			values = append(values, result.Value)
		}

		assertContains(t, visited, "https://example.com/src/app.ts")
		assertContains(t, visited, "https://example.com/src/relative.js")
		assertContains(t, values, "https://example.com/src/app.ts")
		assertContains(t, values, "https://example.com/src/relative.js")
		assertNotContains(t, visited, "https://example.com/maps/relative.js")
	})

	t.Run("absolute sourceRoot", func(t *testing.T) {
		t.Parallel()

		crawler := newTestCrawler(t, &Configuration{
			Domains:           []string{testDomain},
			IncludeSubdomains: true,
		})

		scope := &scope{
			domains:           []string{testDomain},
			includeSubdomains: true,
		}

		mapURL, err := url.Parse("https://example.com/maps/app.js.map")
		if err != nil {
			t.Fatalf("url.Parse() error = %v", err)
		}

		body := []byte(`{"version":3,"sourceRoot":"https://cdn.example.com/src/","sources":["lib.ts"]}`)

		var visited []string

		visit := func(rawURL string) error {
			visited = append(visited, rawURL)

			return nil
		}

		results := make(chan Result, 128)

		crawler.handleSourceMap(t.Context(), mapURL, visit, body, scope, results)

		close(results)

		assertContains(t, visited, "https://cdn.example.com/src/lib.ts")
	})

	t.Run("nil visit still emits", func(t *testing.T) {
		t.Parallel()

		crawler := newTestCrawler(t, &Configuration{Domains: []string{testDomain}})
		scope := &scope{domains: []string{testDomain}}

		mapURL, err := url.Parse("https://example.com/app.js.map")
		if err != nil {
			t.Fatalf("url.Parse() error = %v", err)
		}

		results := make(chan Result, 128)

		crawler.handleSourceMap(t.Context(), mapURL, nil, []byte(`{"sources":["a.ts"]}`), scope, results)

		close(results)

		var values []string

		for result := range results {
			values = append(values, result.Value)
		}

		assertContains(t, values, "https://example.com/a.ts")
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		crawler := newTestCrawler(t, &Configuration{})
		scope := &scope{domains: []string{testDomain}}

		mapURL, err := url.Parse("https://example.com/app.js.map")
		if err != nil {
			t.Fatalf("url.Parse() error = %v", err)
		}

		var visited []string

		visit := func(rawURL string) error {
			visited = append(visited, rawURL)

			return nil
		}

		results := make(chan Result, 8)

		crawler.handleSourceMap(t.Context(), mapURL, visit, []byte("not json"), scope, results)

		close(results)

		if len(visited) != 0 {
			t.Fatalf("visited = %v, want none", visited)
		}
	})
}

func TestIsSourceMapBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rawPath   string
		mediatype string
		body      string
		want      bool
	}{
		{name: "map extension", rawPath: "/app.js.map", mediatype: testMediaBinary, body: "", want: true},
		{name: "json with sources", rawPath: "/map", mediatype: mediaTypeJSON, body: `{"sources":[]}`, want: true},
		{name: "json without sources", rawPath: testDataPath, mediatype: mediaTypeJSON, body: `{"items":[]}`, want: false},
		{name: "html body", rawPath: testIndexPath, mediatype: mediaTypeHTML, body: `<html></html>`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isSourceMapBody(tt.rawPath, tt.mediatype, []byte(tt.body)); got != tt.want {
				t.Fatalf("isSourceMapBody() = %v, want %v", got, tt.want)
			}
		})
	}
}
