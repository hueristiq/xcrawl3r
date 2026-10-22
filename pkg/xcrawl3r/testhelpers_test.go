//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"slices"
	"testing"
	"time"
)

const (
	testDomain         = "example.com"
	testPathURL        = "https://example.com/path"
	testRootURL        = "https://example.com/"
	testSubPathURL     = "https://sub.example.com/path"
	testInvalidURL     = "https://example.com/%zz"
	testSourceRootURL  = "https://example.com/src/"
	testMediaBinary    = "application/octet-stream"
	testMediaPlain     = "text/plain"
	testPunycodeDomain = "xn--mnchen-3ya.de"
	testIndexPath      = "/index.html"
	testDataPath       = "/data"
	testNextPath       = "/next"
)

func newTestCrawler(t *testing.T, cfg *Configuration) *Crawler {
	t.Helper()

	config := Configuration{}

	if cfg != nil {
		config = *cfg
	}

	if config.Timeout == 0 {
		config.Timeout = time.Second
	}

	if config.Parallelism == 0 {
		config.Parallelism = 1
	}

	crawler, err := New(&config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return crawler
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()

	if !slices.Contains(values, want) {
		t.Fatalf("values = %v, want to contain %q", values, want)
	}
}

func assertNotContains(t *testing.T, values []string, want string) {
	t.Helper()

	if slices.Contains(values, want) {
		t.Fatalf("values = %v, want not to contain %q", values, want)
	}
}
