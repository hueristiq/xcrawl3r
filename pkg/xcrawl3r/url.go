package xcrawl3r

import (
	"net"
	"net/url"
	"strings"

	hqgourlparser "github.com/hueristiq/hq-lib-url-go/parser"
	"golang.org/x/net/idna"
)

const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

var up = hqgourlparser.New(hqgourlparser.WithDefaultScheme("https"))

// normalizeURL canonicalizes rawURL: lowercases the host, strips the scheme's
// default port, gives host-only URLs the root path, drops the fragment, and
// decodes percent-escapes only for RFC 3986 unreserved bytes. It is applied to
// both emitted and deduplicated URLs.
func normalizeURL(rawURL string) (normalized string) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.Host = canonicalHost(parsedURL)

	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	parsedURL.Fragment = ""

	if parsedURL.RawQuery == "" {
		parsedURL.ForceQuery = false
	}

	normalized = parsedURL.String()

	normalized = decodePercentEscapes(normalized)

	return
}

// canonicalHost lowercases the host and removes a default port that matches
// the URL's scheme. IPv6 literals keep their brackets.
func canonicalHost(parsedURL *url.URL) (host string) {
	hostname := strings.ToLower(parsedURL.Hostname())
	port := parsedURL.Port()

	switch {
	case parsedURL.Scheme == schemeHTTP && port == "80":
		port = ""
	case parsedURL.Scheme == schemeHTTPS && port == "443":
		port = ""
	}

	if port != "" {
		return net.JoinHostPort(hostname, port)
	}

	if strings.Contains(hostname, ":") {
		return "[" + hostname + "]"
	}

	return hostname
}

func decodePercentEscapes(rawURL string) (decoded string) {
	pathPart, queryPart, hasQuery := strings.Cut(rawURL, "?")

	decoded = decodeEscapes(pathPart, keepEscape)

	if hasQuery {
		decoded += "?" + decodeEscapes(queryPart, keepEscape)
	}

	return
}

func decodeEscapes(s string, keep func(byte) bool) (decoded string) {
	var builder strings.Builder

	builder.Grow(len(s))

	for index := 0; index < len(s); {
		if index+2 < len(s) && s[index] == '%' && isHexDigit(s[index+1]) && isHexDigit(s[index+2]) {
			value := hexValue(s[index+1])<<4 | hexValue(s[index+2])

			if keep(value) {
				builder.WriteString(s[index : index+3])
			} else {
				builder.WriteByte(value)
			}

			index += 3

			continue
		}

		builder.WriteByte(s[index])

		index++
	}

	decoded = builder.String()

	return
}

// keepEscape reports whether a percent-encoded byte must stay encoded. Only
// the RFC 3986 unreserved bytes are decoded; a dot deliberately stays encoded
// so that "%2e%2e" never collapses into a parent-directory segment.
func keepEscape(b byte) (keep bool) {
	switch {
	case b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z', b >= '0' && b <= '9':
		return false
	case b == '-', b == '_', b == '~':
		return false
	}

	return true
}

func isHexDigit(b byte) (is bool) {
	switch {
	case b >= '0' && b <= '9', b >= 'a' && b <= 'f', b >= 'A' && b <= 'F':
		return true
	}

	return
}

func hexValue(b byte) (value byte) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	default:
		return b - 'A' + 10
	}
}

// minifiedCompanion returns the non-minified counterpart of rawURL when its
// path contains a ".min." segment. Only the path is rewritten; the host and
// query are preserved.
func minifiedCompanion(rawURL string) (companion string, ok bool) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return
	}

	if !strings.Contains(parsedURL.Path, ".min.") {
		return
	}

	parsedURL.Path = strings.ReplaceAll(parsedURL.Path, ".min.", ".")
	parsedURL.RawPath = ""

	companion = parsedURL.String()
	ok = true

	return
}

// resolveReference resolves reference against base and returns an absolute
// URL string. Empty and fragment-only references yield "".
func resolveReference(base *url.URL, reference string) (resolved string) {
	if reference == "" || strings.HasPrefix(reference, "#") {
		return
	}

	parsedReference, err := url.Parse(reference)
	if err != nil {
		return
	}

	if base == nil || parsedReference.IsAbs() {
		return parsedReference.String()
	}

	return base.ResolveReference(parsedReference).String()
}

// toASCIIHost converts host to its lowercase ASCII (punycode) form so
// configured unicode domains and parsed hosts compare consistently.
func toASCIIHost(host string) (ascii string) {
	host = strings.ToLower(strings.TrimSuffix(host, "."))

	ascii, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return host
	}

	return strings.ToLower(ascii)
}
