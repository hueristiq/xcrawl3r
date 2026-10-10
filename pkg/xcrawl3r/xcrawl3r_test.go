package xcrawl3r_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
)

func TestCrawl(t *testing.T) {
	t.Parallel()

	var requested sync.Map

	record := func(path string) {
		requested.Store(path, true)
	}

	var server *httptest.Server

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "text/html")

		fmt.Fprintf(w, `<html><body>`+
			`<a href="/page2">p2</a>`+
			`<a href="/page2#top">top</a>`+
			`<a href="https://out-of-scope.example.com/x">x</a>`+
			`<script src="/app.min.js"></script>`+
			`<script>var u = "%s/inline";</script>`+
			`</body></html>`, server.URL)
	})

	mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "text/html")

		fmt.Fprint(w, `<html><body><a href="/">home</a><a href="/deep">deep</a></body></html>`)
	})

	mux.HandleFunc("/app.min.js", func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "application/javascript")

		fmt.Fprintf(w, `var u = "%s/from-js";`, server.URL)
	})

	mux.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "application/javascript")

		fmt.Fprintf(w, `var u = "%s/from-unmin-js";`, server.URL)
	})

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		fmt.Fprintf(w, "User-agent: *\nDisallow: /admin\nDisallow: /private/*\nSitemap: %s/sitemap.xml\n", server.URL)
	})

	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "application/xml")

		fmt.Fprintf(w, `<urlset><url><loc>%s/from-sitemap</loc></url></urlset>`, server.URL)
	})

	server = httptest.NewServer(mux)
	defer server.Close()

	crawler, err := xcrawl3r.New(
		xcrawl3r.WithDepth(2),
		xcrawl3r.WithParallelism(5),
		xcrawl3r.WithTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("xcrawl3r.New() error = %v", err)
	}

	got := make(map[string]int)

	for result := range crawler.Crawl(t.Context(), server.URL) {
		if result.Type == xcrawl3r.ResultError {
			t.Errorf("unexpected error result: %v", result.Error)

			continue
		}

		got[result.Value]++
	}

	expected := []string{
		"/",
		"/page2",
		"/deep",
		"/app.min.js",
		"/inline",
		"/from-js",
		"/from-unmin-js",
		"/from-sitemap",
		"/admin",
		"/private/",
	}

	for _, path := range expected {
		if got[server.URL+path] == 0 {
			t.Errorf("expected %s to be emitted", server.URL+path)
		}
	}

	for URL, count := range got {
		if count > 1 {
			t.Errorf("%s emitted %d times, want 1", URL, count)
		}

		if URL == server.URL+"/page2#top" {
			t.Error("fragment variant should have been deduplicated")
		}

		if URL == server.URL+"/app.js" {
			t.Error("unminified companion should be visited, not emitted")
		}

		if URL == server.URL+"/sitemap.xml" {
			t.Error("sitemap from robots.txt should be visited, not emitted")
		}
	}

	if _, ok := requested.Load("/deep"); ok {
		t.Error("/deep is beyond depth 2 and should not have been requested")
	}

	for _, path := range []string{"/page2", "/app.js", "/sitemap.xml", "/admin", "/private/"} {
		if _, ok := requested.Load(path); !ok {
			t.Errorf("expected %s to be requested", path)
		}
	}
}

func TestCrawlDeduplicatesAcrossTargets(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		fmt.Fprint(w, `<html><body><a href="/page">p</a></body></html>`)
	}))
	defer server.Close()

	crawler, err := xcrawl3r.New()
	if err != nil {
		t.Fatalf("xcrawl3r.New() error = %v", err)
	}

	for result := range crawler.Crawl(t.Context(), server.URL) {
		if result.Type == xcrawl3r.ResultError {
			t.Errorf("unexpected error result: %v", result.Error)
		}
	}

	for result := range crawler.Crawl(t.Context(), server.URL+"/") {
		if result.Type == xcrawl3r.ResultURL {
			t.Errorf("unexpected duplicate result: %s", result.Value)
		}
	}
}
