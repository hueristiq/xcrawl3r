package xcrawl3r_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
)

func TestCrawl(t *testing.T) {
	t.Parallel()

	const (
		page2Path  = "/page2"
		appMinJS   = "/app.min.js"
		sitemapXML = "/sitemap.xml"
		adminPath  = "/admin"
	)

	var requested sync.Map

	record := func(path string) {
		requested.Store(path, true)
	}

	var server *httptest.Server

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "text/html")

		if r.URL.Path != "/" {
			_, _ = fmt.Fprint(w, `<html><body></body></html>`)

			return
		}

		_, _ = fmt.Fprintf(w, `<html><body>`+
			`<a href="/page2">p2</a>`+
			`<a href="/page2#top">top</a>`+
			`<a href="https://out-of-scope.example.com/x">x</a>`+
			`<script src="/app.min.js"></script>`+
			`<script>var u = "%s/inline";</script>`+
			`</body></html>`, server.URL)
	})

	mux.HandleFunc(page2Path, func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "text/html")

		_, _ = fmt.Fprint(w, `<html><body><a href="/">home</a><a href="/deep">deep</a></body></html>`)
	})

	mux.HandleFunc(appMinJS, func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "application/javascript")

		_, _ = fmt.Fprintf(w, `var u = "%s/from-js";`, server.URL)
	})

	mux.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "application/javascript")

		_, _ = fmt.Fprintf(w, `var u = "%s/from-unmin-js";`, server.URL)
	})

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		_, _ = fmt.Fprintf(w, "User-agent: *\nDisallow: /admin\nDisallow: /private/*\nSitemap: %s/sitemap.xml\n", server.URL)
	})

	mux.HandleFunc(sitemapXML, func(w http.ResponseWriter, r *http.Request) {
		record(r.URL.Path)

		w.Header().Set("Content-Type", "application/xml")

		_, _ = fmt.Fprintf(w, `<urlset><url><loc>%s/from-sitemap</loc></url></urlset>`, server.URL)
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
	provenance := make(map[string]xcrawl3r.Result)

	for result := range crawler.Crawl(t.Context(), server.URL) {
		if result.Type == xcrawl3r.ResultError {
			t.Errorf("unexpected error result: %v", result.Error)

			continue
		}

		got[result.Value]++

		if _, ok := provenance[result.Value]; !ok {
			provenance[result.Value] = result
		}
	}

	expected := []string{
		"/",
		page2Path,
		"/deep",
		appMinJS,
		"/inline",
		"/from-js",
		"/from-unmin-js",
		"/from-sitemap",
		adminPath,
		"/private/",
	}

	for _, path := range expected {
		if got[server.URL+path] == 0 {
			t.Errorf("expected %s to be emitted", server.URL+path)
		}
	}

	for _, junk := range []string{"/loc", "/url", "/urlset"} {
		if got[server.URL+junk] > 0 {
			t.Errorf("%s emitted from closing XML tags, want it suppressed", server.URL+junk)
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

		if URL == server.URL+sitemapXML {
			t.Error("sitemap from robots.txt should be visited, not emitted")
		}
	}

	if _, ok := requested.Load("/deep"); ok {
		t.Error("/deep is beyond depth 2 and should not have been requested")
	}

	for _, path := range []string{page2Path, "/app.js", sitemapXML, adminPath, "/private/"} {
		if _, ok := requested.Load(path); !ok {
			t.Errorf("expected %s to be requested", path)
		}
	}

	provenanceChecks := []struct {
		path         string
		via          string
		sourceSuffix string
	}{
		{page2Path, "href", "/"},
		{appMinJS, "src", "/"},
		{"/inline", "regex", "/"},
		{"/from-js", "regex", appMinJS},
		{"/from-sitemap", "sitemap", sitemapXML},
		{adminPath, "robots", "/robots.txt"},
	}

	for _, check := range provenanceChecks {
		result, ok := provenance[server.URL+check.path]
		if !ok {
			t.Errorf("expected %s to be emitted", server.URL+check.path)

			continue
		}

		if result.Via != check.via {
			t.Errorf("%s via = %q, want %q", check.path, result.Via, check.via)
		}

		if !strings.HasSuffix(result.Source, check.sourceSuffix) {
			t.Errorf("%s source = %q, want suffix %q", check.path, result.Source, check.sourceSuffix)
		}
	}
}

func TestCrawlJavaScriptTagJunkSuppressed(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		_, _ = fmt.Fprint(w, `<html><body><script src="/app.js"></script></body></html>`)
	})

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "User-agent: *\n")
	})

	mux.HandleFunc("/app.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")

		_, _ = fmt.Fprint(w, `const el = document.getElementById("x");`+
			`el.innerHTML = "</div><span>hi</span>";`+
			`const escaped = "\u003c/div\u003e";`+
			`const api = "/api/v1/users";`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	crawler, err := xcrawl3r.New(
		xcrawl3r.WithDepth(2),
		xcrawl3r.WithParallelism(4),
		xcrawl3r.WithTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("xcrawl3r.New() error = %v", err)
	}

	got := make(map[string]bool)

	for result := range crawler.Crawl(t.Context(), server.URL) {
		if result.Type == xcrawl3r.ResultError {
			t.Errorf("unexpected error result: %v", result.Error)

			continue
		}

		got[result.Value] = true
	}

	if !got[server.URL+"/api/v1/users"] {
		t.Errorf("expected relative URL %s/api/v1/users to be emitted", server.URL)
	}

	for _, junk := range []string{"/div", "/span"} {
		if got[server.URL+junk] {
			t.Errorf("%s emitted from closing HTML tags, want it suppressed", server.URL+junk)
		}
	}
}

func TestNewValidatesOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opt  xcrawl3r.OptionFunc
	}{
		{"zero timeout", xcrawl3r.WithTimeout(0)},
		{"negative timeout", xcrawl3r.WithTimeout(-time.Second)},
		{"zero parallelism", xcrawl3r.WithParallelism(0)},
		{"negative parallelism", xcrawl3r.WithParallelism(-1)},
		{"negative depth", xcrawl3r.WithDepth(-1)},
		{"negative max pages", xcrawl3r.WithMaxPages(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := xcrawl3r.New(tt.opt); err == nil {
				t.Error("New() expected error, got nil")
			}
		})
	}
}

func TestCrawlDeduplicatesAcrossTargets(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		_, _ = fmt.Fprint(w, `<html><body><a href="/page">p</a></body></html>`)
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

func TestCrawlConcurrentTargets(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = fmt.Fprint(w, "User-agent: *\n")

			return
		}

		w.Header().Set("Content-Type", "text/html")

		if r.URL.Path == "/" {
			_, _ = fmt.Fprint(w, `<html><body><a href="/page">p</a></body></html>`)

			return
		}

		_, _ = fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer server.Close()

	crawler, err := xcrawl3r.New(
		xcrawl3r.WithDepth(1),
		xcrawl3r.WithParallelism(4),
		xcrawl3r.WithTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("xcrawl3r.New() error = %v", err)
	}

	// The shared colly storage and seen-set must be safe for concurrent
	// Crawl calls; this exercises the lazy-Init data race fixed in New().
	const workers = 5

	var emitted sync.Map

	wg := &sync.WaitGroup{}

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for result := range crawler.Crawl(t.Context(), server.URL) {
				if result.Type == xcrawl3r.ResultError {
					t.Errorf("unexpected error result: %v", result.Error)

					continue
				}

				if _, duplicate := emitted.LoadOrStore(result.Value, struct{}{}); duplicate {
					t.Errorf("duplicate result across concurrent crawls: %s", result.Value)
				}
			}
		}()
	}

	wg.Wait()

	if _, ok := emitted.Load(server.URL + "/page"); !ok {
		t.Errorf("expected %s/page to be emitted", server.URL)
	}
}

func TestCrawlMaxPages(t *testing.T) {
	t.Parallel()

	var requested sync.Map

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requested.Store(r.URL.Path, true)

		w.Header().Set("Content-Type", "text/html")

		for i := range 20 {
			_, _ = fmt.Fprintf(w, `<a href="/p%d">p%d</a>`, i, i)
		}
	})

	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		requested.Store(r.URL.Path, true)

		_, _ = fmt.Fprint(w, "User-agent: *\n")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	const maxPages = 12

	crawler, err := xcrawl3r.New(
		xcrawl3r.WithDepth(2),
		xcrawl3r.WithParallelism(1),
		xcrawl3r.WithMaxPages(maxPages),
		xcrawl3r.WithTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("xcrawl3r.New() error = %v", err)
	}

	for result := range crawler.Crawl(t.Context(), server.URL) {
		if result.Type == xcrawl3r.ResultError {
			t.Errorf("unexpected error result: %v", result.Error)
		}
	}

	count := 0

	requested.Range(func(_, _ any) bool {
		count++

		return true
	})

	if count != maxPages {
		t.Errorf("requested %d pages, want %d", count, maxPages)
	}
}

func TestCrawlDuration(t *testing.T) {
	t.Parallel()

	var requested sync.Map

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested.Store(r.URL.Path, true)

		if r.URL.Path != "/" {
			time.Sleep(20 * time.Millisecond)
		}

		w.Header().Set("Content-Type", "text/html")

		if r.URL.Path == "/" {
			for i := range 200 {
				_, _ = fmt.Fprintf(w, `<a href="/p%d">p%d</a>`, i, i)
			}

			return
		}

		_, _ = fmt.Fprint(w, `<html><body></body></html>`)
	}))
	defer server.Close()

	crawler, err := xcrawl3r.New(
		xcrawl3r.WithDepth(2),
		xcrawl3r.WithParallelism(10),
		xcrawl3r.WithCrawlDuration(300*time.Millisecond),
		xcrawl3r.WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("xcrawl3r.New() error = %v", err)
	}

	start := time.Now()

	for result := range crawler.Crawl(t.Context(), server.URL) {
		if result.Type == xcrawl3r.ResultError {
			t.Errorf("unexpected error result: %v", result.Error)
		}
	}

	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("crawl took %v, want it capped by crawl duration", elapsed)
	}

	count := 0

	requested.Range(func(_, _ any) bool {
		count++

		return true
	})

	if count >= 200 {
		t.Errorf("requested %d pages, crawl duration should have cut it short of 200", count)
	}
}
