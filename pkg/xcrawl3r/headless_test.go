package xcrawl3r_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
)

func headlessAvailable() (available bool) {
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}

	return
}

func TestCrawlHeadless(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping headless test in short mode")
	}

	if !headlessAvailable() {
		t.Skip("no Chrome/Chromium binary found")
	}

	t.Parallel()

	var requested sync.Map

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requested.Store(r.URL.Path, true)

		w.Header().Set("Content-Type", "text/html")

		_, _ = fmt.Fprint(w, `<html><body><div id="root"></div>`+
			`<script>document.getElementById("root").innerHTML = '<a href="/js-page">spa</a>';</script>`+
			`</body></html>`)
	})

	mux.HandleFunc("/js-page", func(w http.ResponseWriter, r *http.Request) {
		requested.Store(r.URL.Path, true)

		w.Header().Set("Content-Type", "text/html")

		_, _ = fmt.Fprint(w, `<html><body></body></html>`)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	crawler, err := xcrawl3r.New(
		xcrawl3r.WithDepth(2),
		xcrawl3r.WithParallelism(2),
		xcrawl3r.WithTimeout(10*time.Second),
		xcrawl3r.WithHeadless(true),
		xcrawl3r.WithHeadlessTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("xcrawl3r.New() error = %v", err)
	}

	defer crawler.Close()

	if err := crawler.Validate(); err != nil {
		t.Fatalf("crawler.Validate() error = %v", err)
	}

	var found bool

	for result := range crawler.Crawl(t.Context(), server.URL) {
		if result.Type == xcrawl3r.ResultError {
			t.Errorf("unexpected error result: %v", result.Error)

			continue
		}

		if result.Value == server.URL+"/js-page" {
			found = true

			if result.Via != "headless" {
				t.Errorf("via = %q, want %q", result.Via, "headless")
			}
		}
	}

	if !found {
		t.Error("expected JS-injected link /js-page to be found via headless rendering")
	}

	if _, ok := requested.Load("/js-page"); !ok {
		t.Error("expected /js-page to be requested")
	}
}
