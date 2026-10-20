package xcrawl3r

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	shapeNumericSegmentRegex = regexp.MustCompile(`^\d+$`)
	shapeUUIDSegmentRegex    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	shapeHexSegmentRegex     = regexp.MustCompile(`^[0-9a-f]{16,}$`)
	shapeDateSegmentRegex    = regexp.MustCompile(`^\d{4}-\d{2}(-\d{2})?$`)
)

// normalizeURL canonicalizes rawURL: lowercases the host, strips default
// ports, gives host-only URLs the root path, collapses repeated slashes,
// drops the fragment, and decodes percent-escapes that do not change the
// URL's structure. It is applied to both emitted and deduplicated URLs.
func normalizeURL(rawURL string) (normalized string) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	parsedURL.Host = strings.ToLower(parsedURL.Host)

	if port := parsedURL.Port(); port == "80" || port == "443" {
		parsedURL.Host = parsedURL.Hostname()
	}

	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	path := strings.ReplaceAll(parsedURL.Path, "//", "/")

	if path != parsedURL.Path {
		parsedURL.Path = path
		parsedURL.RawPath = ""
	}

	parsedURL.Fragment = ""

	if parsedURL.RawQuery == "" {
		parsedURL.ForceQuery = false
	}

	normalized = parsedURL.String()

	normalized = decodePercentEscapes(normalized)

	return
}

func decodePercentEscapes(rawURL string) (decoded string) {
	pathPart, queryPart, hasQuery := strings.Cut(rawURL, "?")

	decoded = decodeEscapes(pathPart, keepPathEscape)

	if hasQuery {
		decoded += "?" + decodeEscapes(queryPart, keepQueryEscape)
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

func keepPathEscape(b byte) (keep bool) {
	if b < 0x20 || b == 0x7F {
		return true
	}

	switch b {
	case '/', '?', '#', '%', ' ':
		return true
	}

	return
}

func keepQueryEscape(b byte) (keep bool) {
	if b < 0x20 || b == 0x7F {
		return true
	}

	switch b {
	case '&', '#', '%', '+', '=':
		return true
	}

	return
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

func (c *Crawler) dedupeKey(rawURL string) (key string) {
	key = normalizeURL(rawURL)

	if strings.HasPrefix(key, "http://") {
		key = "https://" + key[len("http://"):]
	}

	if c.cfg.IgnoreQuery {
		key = stripQuery(key)
	}

	if c.cfg.ShapeDedup {
		key = stripShape(key)
	}

	return
}

func stripQuery(rawURL string) (stripped string) {
	stripped = rawURL

	if index := strings.IndexByte(stripped, '?'); index >= 0 {
		stripped = stripped[:index]
	}

	return
}

func stripShape(rawURL string) (stripped string) {
	stripped = rawURL

	schemeEnd := strings.Index(stripped, "://")
	if schemeEnd < 0 {
		return
	}

	authority := stripped[schemeEnd+3:]

	pathRel := strings.IndexByte(authority, '/')
	if pathRel < 0 {
		return
	}

	pathStart := schemeEnd + 3 + pathRel
	pathEnd := len(stripped)

	if index := strings.IndexByte(stripped[pathStart:], '?'); index >= 0 {
		pathEnd = pathStart + index
	}

	path := stripped[pathStart:pathEnd]

	segments := strings.Split(path, "/")

	changed := false

	for i, segment := range segments {
		switch {
		case shapeNumericSegmentRegex.MatchString(segment),
			shapeUUIDSegmentRegex.MatchString(segment),
			shapeHexSegmentRegex.MatchString(segment),
			shapeDateSegmentRegex.MatchString(segment):
			segments[i] = "{}"

			changed = true
		}
	}

	if !changed {
		return
	}

	stripped = stripped[:pathStart] + strings.Join(segments, "/") + stripped[pathEnd:]

	return
}
