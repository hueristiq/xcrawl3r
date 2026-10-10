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

func normalizeURL(rawURL string, stripQuery, stripShape bool) (normalized string) {
	normalized = rawURL

	if index := strings.IndexByte(normalized, '#'); index >= 0 {
		normalized = normalized[:index]
	}

	if stripQuery {
		if index := strings.IndexByte(normalized, '?'); index >= 0 {
			normalized = normalized[:index]
		}
	}

	if !stripShape {
		return
	}

	schemeEnd := strings.Index(normalized, "://")
	if schemeEnd < 0 {
		return
	}

	authority := normalized[schemeEnd+3:]

	pathRel := strings.IndexByte(authority, '/')
	if pathRel < 0 {
		return
	}

	pathStart := schemeEnd + 3 + pathRel
	pathEnd := len(normalized)

	if index := strings.IndexByte(normalized[pathStart:], '?'); index >= 0 {
		pathEnd = pathStart + index
	}

	path := normalized[pathStart:pathEnd]

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

	normalized = normalized[:pathStart] + strings.Join(segments, "/") + normalized[pathEnd:]

	return
}
