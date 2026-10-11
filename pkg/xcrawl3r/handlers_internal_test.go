package xcrawl3r

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/gocolly/colly/v2"
)

func TestVisitCountsOnlySuccessfulVisits(t *testing.T) {
	t.Parallel()

	crawler := &Crawler{opts: defaultOptions()}

	s := &scope{maxPages: 3}

	results := make(chan Result, 8)

	var attempts atomic.Int32

	fail := func() error {
		attempts.Add(1)

		return &colly.AlreadyVisitedError{}
	}

	succeed := func() error {
		attempts.Add(1)

		return nil
	}

	// Failed visits must not consume the page budget.
	for range 3 {
		crawler.visit(t.Context(), "https://example.com/seen-before", results, s, fail)
	}

	if got := s.requestCount.Load(); got != 0 {
		t.Errorf("request count = %d after failed visits, want 0", got)
	}

	// Successful visits consume the budget up to maxPages.
	for i := range 3 {
		crawler.visit(t.Context(), fmt.Sprintf("https://example.com/ok-%d", i), results, s, succeed)
	}

	if got := s.requestCount.Load(); got != 3 {
		t.Errorf("request count = %d after successful visits, want 3", got)
	}

	// The exhausted budget refuses further visits without calling do.
	crawler.visit(t.Context(), "https://example.com/blocked", results, s, succeed)

	if got := attempts.Load(); got != 6 {
		t.Errorf("do() calls = %d, want 6", got)
	}
}

func TestIsSitemapPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rawPath string
		want    bool
	}{
		{"/sitemap.xml", true},
		{"/sitemap.xml.gz", true},
		{"/sitemap_index.xml.gz", true},
		{"/SITEMAP.XML", true},
		{"/wp-sitemap.xml", true},
		{"/foo.xml", false},
		{"/sitemap.html", false},
		{"/robots.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.rawPath, func(t *testing.T) {
			t.Parallel()

			if got := isSitemapPath(tt.rawPath); got != tt.want {
				t.Errorf("isSitemapPath(%q) = %v, want %v", tt.rawPath, got, tt.want)
			}
		})
	}
}
