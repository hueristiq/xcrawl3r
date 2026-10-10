package xcrawl3r

import (
	"testing"
)

const (
	exampleDomain = "example.com"
	loopbackIP    = "127.0.0.1"
)

func TestScopeAllows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		domains           []string
		includeSubdomains bool
		rawURL            string
		want              bool
	}{
		{"apex match", []string{exampleDomain}, false, "https://example.com", true},
		{"apex match with port", []string{exampleDomain}, false, "https://example.com:8443/path?q=1", true},
		{"apex match case insensitive", []string{exampleDomain}, false, "https://EXAMPLE.com/", true},
		{"http scheme allowed", []string{exampleDomain}, false, "http://example.com", true},
		{"subdomain rejected", []string{exampleDomain}, false, "https://sub.example.com", false},
		{"subdomain allowed", []string{exampleDomain}, true, "https://sub.example.com", true},
		{"deep subdomain allowed", []string{exampleDomain}, true, "https://a.b.example.com", true},
		{"suffix attack rejected", []string{exampleDomain}, true, "https://example.com.evil.com", false},
		{"prefix attack rejected", []string{exampleDomain}, true, "https://evil-example.com", false},
		{"userinfo attack rejected", []string{exampleDomain}, true, "https://www.example.com@evil.com", false},
		{"query does not leak scope", []string{exampleDomain}, false, "https://evil.com/?next=https://example.com", false},
		{"query URL ignored", []string{exampleDomain}, false, "https://example.com/?next=https://evil.com", true},
		{"fragment does not matter", []string{exampleDomain}, false, "https://example.com/#frag", true},
		{"non-http scheme rejected", []string{exampleDomain}, false, "ftp://example.com", false},
		{"invalid URL rejected", []string{exampleDomain}, false, "%", false},
		{"ip host match", []string{loopbackIP}, false, "http://127.0.0.1:8080/", true},
		{"ip host mismatch", []string{loopbackIP}, false, "http://127.0.0.2:8080/", false},
		{"ip host subdomain no match", []string{loopbackIP}, true, "http://127.0.0.2/", false},
		{"multiple domains", []string{exampleDomain, "example.org"}, false, "https://example.org/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := &scope{
				domains:           tt.domains,
				includeSubdomains: tt.includeSubdomains,
			}

			if got := s.allows(tt.rawURL); got != tt.want {
				t.Errorf("allows(%q) = %v, want %v", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestScopeFor(t *testing.T) {
	t.Parallel()

	parsedTargetURL, err := up.Parse("https://WWW.Example.com/path")
	if err != nil {
		t.Fatalf("up.Parse() error = %v", err)
	}

	t.Run("defaults to seed host", func(t *testing.T) {
		t.Parallel()

		crawler := &Crawler{opts: defaultOptions()}

		s := crawler.scopeFor(parsedTargetURL)

		if !s.allows("https://www.example.com/other") {
			t.Error("seed host should be allowed")
		}

		if s.allows("https://example.com/") {
			t.Error("apex should not be allowed when seed host has a subdomain")
		}

		if s.allows("https://sub.example.com/") {
			t.Error("sibling subdomain should not be allowed")
		}
	})

	t.Run("configured domains replace seed host", func(t *testing.T) {
		t.Parallel()

		crawler := &Crawler{opts: defaultOptions()}
		crawler.opts.domains = []string{"Example.org"}

		s := crawler.scopeFor(parsedTargetURL)

		if !s.allows("https://example.org/") {
			t.Error("configured domain should be allowed")
		}

		if s.allows("https://www.example.com/") {
			t.Error("seed host should not be allowed when domains are configured")
		}
	})
}
