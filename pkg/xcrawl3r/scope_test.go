//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestScopeAllows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		rawURL            string
		domains           []string
		includeSubdomains bool
		want              bool
	}{
		{
			name:    "exact host",
			rawURL:  testPathURL,
			domains: []string{testDomain},
			want:    true,
		},
		{
			name:    "subdomain excluded by default",
			rawURL:  testSubPathURL,
			domains: []string{testDomain},
			want:    false,
		},
		{
			name:              "subdomain included",
			rawURL:            testSubPathURL,
			domains:           []string{testDomain},
			includeSubdomains: true,
			want:              true,
		},
		{
			name:              "unrelated host",
			rawURL:            "https://other.com/path",
			domains:           []string{testDomain},
			includeSubdomains: true,
			want:              false,
		},
		{
			name:              "suffix trap",
			rawURL:            "https://evilexample.com/path",
			domains:           []string{testDomain},
			includeSubdomains: true,
			want:              false,
		},
		{
			name:    "non-http scheme",
			rawURL:  "ftp://example.com/path",
			domains: []string{testDomain},
			want:    false,
		},
		{
			name:    "uppercase host",
			rawURL:  "https://EXAMPLE.COM/path",
			domains: []string{testDomain},
			want:    true,
		},
		{
			name:    "trailing dot host",
			rawURL:  "https://example.com./path",
			domains: []string{testDomain},
			want:    true,
		},
		{
			name:    "unicode host against punycode domain",
			rawURL:  "https://münchen.de/path",
			domains: []string{testPunycodeDomain},
			want:    true,
		},
		{
			name:    "punycode host against normalized unicode domain",
			rawURL:  "https://xn--mnchen-3ya.de/path",
			domains: []string{testPunycodeDomain},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &scope{
				domains:           tt.domains,
				includeSubdomains: tt.includeSubdomains,
			}

			if got := s.allows(tt.rawURL); got != tt.want {
				t.Fatalf("allows(%q) = %v, want %v", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestScopeForNormalizesDomains(t *testing.T) {
	t.Parallel()

	crawler := &Crawler{
		cfg: &Configuration{
			Domains: []string{"münchen.de"},
		},
	}

	target, err := up.Parse(testRootURL)
	if err != nil {
		t.Fatalf("up.Parse() error = %v", err)
	}

	s := crawler.scopeFor(target)

	if !s.allows("https://xn--mnchen-3ya.de/path") {
		t.Fatal("allows(punycode host) = false, want true after scopeFor normalization")
	}

	if !s.allows("https://münchen.de/path") {
		t.Fatal("allows(unicode host) = false, want true after scopeFor normalization")
	}
}

func TestScopeRegistrableAllows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawURL  string
		domains []string
		want    bool
	}{
		{
			name:    "sibling subdomain",
			rawURL:  testSubPathURL,
			domains: []string{testDomain},
			want:    true,
		},
		{
			name:    "unrelated registrable domain",
			rawURL:  "https://other.com/path",
			domains: []string{testDomain},
			want:    false,
		},
		{
			name:    "ip literal",
			rawURL:  "http://127.0.0.1/path",
			domains: []string{testDomain},
			want:    false,
		},
		{
			name:    "normalized unicode domain",
			rawURL:  "https://sub.xn--mnchen-3ya.de/path",
			domains: []string{testPunycodeDomain},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &scope{domains: tt.domains}

			if got := s.registrableAllows(tt.rawURL); got != tt.want {
				t.Fatalf("registrableAllows(%q) = %v, want %v", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestScopeWiden(t *testing.T) {
	t.Parallel()

	s := &scope{domains: []string{testDomain}}

	if s.widen(testDomain) {
		t.Fatal("widen(existing domain) = true, want false")
	}

	if !s.widen("sub.example.com") {
		t.Fatal("widen(new host) = false, want true")
	}

	if !s.allows(testSubPathURL) {
		t.Fatal("allows(widened host) = false, want true")
	}

	if s.widen("sub.example.com") {
		t.Fatal("widen(already widened host) = true, want false")
	}

	withSubdomains := &scope{
		domains:           []string{testDomain},
		includeSubdomains: true,
	}

	if withSubdomains.widen("sub.example.com") {
		t.Fatal("widen(subdomain already in scope) = true, want false")
	}

	unicode := &scope{domains: []string{testDomain}}

	if !unicode.widen("MÜNCHEN.DE") {
		t.Fatal("widen(unicode host) = false, want true")
	}

	if !unicode.allows("https://xn--mnchen-3ya.de/path") {
		t.Fatal("allows(punycode host after unicode widen) = false, want true")
	}
}

func TestScopeChargePage(t *testing.T) {
	t.Parallel()

	t.Run("unlimited", func(t *testing.T) {
		t.Parallel()

		s := &scope{maxPages: 0}

		for range 5 {
			if !s.chargePage() {
				t.Fatal("chargePage() = false with no limit, want true")
			}
		}
	})

	t.Run("limited", func(t *testing.T) {
		t.Parallel()

		s := &scope{maxPages: 2}

		if !s.chargePage() {
			t.Fatal("first chargePage() = false, want true")
		}

		if !s.chargePage() {
			t.Fatal("second chargePage() = false, want true")
		}

		if s.chargePage() {
			t.Fatal("third chargePage() = true, want false")
		}

		if s.chargePage() {
			t.Fatal("fourth chargePage() = true, want false")
		}
	})

	t.Run("concurrent", func(t *testing.T) {
		t.Parallel()

		s := &scope{maxPages: 50}

		var allowed atomic.Int32

		var wg sync.WaitGroup

		for range 100 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				if s.chargePage() {
					allowed.Add(1)
				}
			}()
		}

		wg.Wait()

		if got := allowed.Load(); got != 50 {
			t.Fatalf("allowed = %d, want 50", got)
		}
	})
}
