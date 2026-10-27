package xcrawl3r

import (
	"regexp"
	"strings"
)

var (
	shapeNumericSegmentRegex = regexp.MustCompile(`^\d+$`)
	shapeUUIDSegmentRegex    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	shapeHexSegmentRegex     = regexp.MustCompile(`^[0-9a-f]{16,}$`)
	shapeDateSegmentRegex    = regexp.MustCompile(`^\d{4}-\d{2}(-\d{2})?$`)
)

func (c *Crawler) dedupeKey(rawURL string) (key string) {
	key = normalizeURL(rawURL)

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
