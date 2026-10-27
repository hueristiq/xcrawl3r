//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"net/url"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rawURL string
		want   string
	}{
		{
			name:   "lowercases host and strips default http port",
			rawURL: "http://Example.COM:80/Path",
			want:   "http://example.com/Path",
		},
		{
			name:   "strips default https port",
			rawURL: "https://example.com:443",
			want:   testRootURL,
		},
		{
			name:   "keeps non-default port for scheme",
			rawURL: "https://example.com:80/path",
			want:   "https://example.com:80/path",
		},
		{
			name:   "keeps non-default http port",
			rawURL: "http://example.com:8080/path",
			want:   "http://example.com:8080/path",
		},
		{
			name:   "strips default port from ipv6",
			rawURL: "http://[::1]:80/path",
			want:   "http://[::1]/path",
		},
		{
			name:   "keeps non-default ipv6 port",
			rawURL: "https://[::1]:8443/path",
			want:   "https://[::1]:8443/path",
		},
		{
			name:   "adds root path and drops fragment",
			rawURL: "https://example.com#section",
			want:   testRootURL,
		},
		{
			name:   "keeps query",
			rawURL: "https://example.com/path?a=1&b=2#section",
			want:   "https://example.com/path?a=1&b=2",
		},
		{
			name:   "decodes unreserved escape",
			rawURL: "https://example.com/%7euser",
			want:   "https://example.com/~user",
		},
		{
			name:   "keeps encoded slash",
			rawURL: "https://example.com/%2f/x",
			want:   "https://example.com/%2f/x",
		},
		{
			name:   "keeps encoded dot segments",
			rawURL: "https://example.com/%2e%2e/x",
			want:   "https://example.com/%2e%2e/x",
		},
		{
			name:   "keeps repeated slashes",
			rawURL: "https://example.com/foo//bar",
			want:   "https://example.com/foo//bar",
		},
		{
			name:   "preserves scheme",
			rawURL: "http://example.com/path",
			want:   "http://example.com/path",
		},
		{
			name:   "decodes unreserved letter",
			rawURL: "https://example.com/%41",
			want:   "https://example.com/A",
		},
		{
			name:   "leaves invalid escape intact",
			rawURL: testInvalidURL,
			want:   testInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeURL(tt.rawURL); got != tt.want {
				t.Fatalf("normalizeURL(%q) = %q, want %q", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestCanonicalHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		rawURL string
		want   string
	}{
		{name: "http default port", rawURL: "http://example.com:80/", want: testDomain},
		{name: "https default port", rawURL: "https://example.com:443/", want: testDomain},
		{name: "https keeps port 80", rawURL: "https://example.com:80/", want: "example.com:80"},
		{name: "http keeps port 443", rawURL: "http://example.com:443/", want: "example.com:443"},
		{name: "uppercase host", rawURL: "http://Example.COM:8080/", want: "example.com:8080"},
		{name: "ipv6 default port", rawURL: "http://[::1]:80/", want: "[::1]"},
		{name: "ipv6 non-default port", rawURL: "https://[::1]:8443/", want: "[::1]:8443"},
		{name: "ipv6 without port", rawURL: "http://[2001:db8::1]/", want: "[2001:db8::1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsedURL, err := url.Parse(tt.rawURL)
			if err != nil {
				t.Fatalf("url.Parse(%q) error = %v", tt.rawURL, err)
			}

			if got := canonicalHost(parsedURL); got != tt.want {
				t.Fatalf("canonicalHost(%q) = %q, want %q", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestDecodeEscapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "unreserved and reserved", input: "%41%2f%2e%2e%7e%20", want: "A%2f%2e%2e~%20"},
		{name: "invalid escapes", input: "%zz%4", want: "%zz%4"},
		{name: "uppercase unreserved", input: "%7E", want: "~"},
		{name: "hyphen and underscore", input: "%2D%5F", want: "-_"},
		{name: "plain text", input: "plain", want: "plain"},
		{name: "empty input", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := decodeEscapes(tt.input, keepEscape); got != tt.want {
				t.Fatalf("decodeEscapes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecodePercentEscapes(t *testing.T) {
	t.Parallel()

	input := "https://example.com/a%7e?q=%41%2f"
	want := "https://example.com/a~?q=A%2f"

	if got := decodePercentEscapes(input); got != want {
		t.Fatalf("decodePercentEscapes(%q) = %q, want %q", input, got, want)
	}
}

func TestMinifiedCompanion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		rawURL        string
		wantCompanion string
		wantOK        bool
	}{
		{
			name:          "rewrites path and keeps query",
			rawURL:        "https://example.com/app.min.js?v=2",
			wantCompanion: "https://example.com/app.js?v=2",
			wantOK:        true,
		},
		{
			name:          "rewrites every path segment",
			rawURL:        "https://example.com/a.min.b.min.js",
			wantCompanion: "https://example.com/a.b.js",
			wantOK:        true,
		},
		{
			name:   "host segment untouched",
			rawURL: "https://app.min.example.com/app.js",
			wantOK: false,
		},
		{
			name:   "query segment untouched",
			rawURL: "https://example.com/app.js?file=x.min.js",
			wantOK: false,
		},
		{
			name:   "not minified",
			rawURL: "https://example.com/app.js",
			wantOK: false,
		},
		{
			name:   "invalid url",
			rawURL: testInvalidURL,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotCompanion, gotOK := minifiedCompanion(tt.rawURL)
			if gotOK != tt.wantOK {
				t.Fatalf("minifiedCompanion(%q) ok = %v, want %v", tt.rawURL, gotOK, tt.wantOK)
			}

			if gotCompanion != tt.wantCompanion {
				t.Fatalf("minifiedCompanion(%q) = %q, want %q", tt.rawURL, gotCompanion, tt.wantCompanion)
			}
		})
	}
}

func TestResolveReference(t *testing.T) {
	t.Parallel()

	base, err := url.Parse("https://example.com/dir/page.html")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	tests := []struct {
		name      string
		base      *url.URL
		reference string
		want      string
	}{
		{name: "absolute path reference", base: base, reference: "/abs", want: "https://example.com/abs"},
		{name: "relative path", base: base, reference: "rel.js", want: "https://example.com/dir/rel.js"},
		{name: "parent path", base: base, reference: "../up.js", want: "https://example.com/up.js"},
		{name: "absolute url reference", base: base, reference: "https://other.com/x", want: "https://other.com/x"},
		{name: "scheme relative", base: base, reference: "//cdn.example.com/x", want: "https://cdn.example.com/x"},
		{name: "fragment", base: base, reference: "#section", want: ""},
		{name: "empty reference", base: base, reference: "", want: ""},
		{name: "query", base: base, reference: "?q=1", want: "https://example.com/dir/page.html?q=1"},
		{name: "invalid reference", base: base, reference: "%zz", want: ""},
		{name: "nil base", base: nil, reference: "rel", want: "rel"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := resolveReference(tt.base, tt.reference); got != tt.want {
				t.Fatalf("resolveReference(%q) = %q, want %q", tt.reference, got, tt.want)
			}
		})
	}
}

func TestToASCIIHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host string
		want string
	}{
		{host: "MÜNCHEN.DE", want: testPunycodeDomain},
		{host: "Example.COM.", want: testDomain},
		{host: "xn--p1ai", want: "xn--p1ai"},
		{host: "::1", want: "::1"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			t.Parallel()

			if got := toASCIIHost(tt.host); got != tt.want {
				t.Fatalf("toASCIIHost(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}
