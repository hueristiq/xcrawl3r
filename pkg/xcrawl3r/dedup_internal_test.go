package xcrawl3r

import "testing"

const pageURL = "https://example.com/page"

func TestNormalizeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		raw        string
		stripQuery bool
		stripShape bool
		expected   string
	}{
		{
			name:     "strips fragment",
			raw:      pageURL + "#top",
			expected: pageURL,
		},
		{
			name:       "strips query",
			raw:        pageURL + "?a=1&b=2",
			stripQuery: true,
			expected:   pageURL,
		},
		{
			name:     "keeps query by default",
			raw:      pageURL + "?a=1",
			expected: pageURL + "?a=1",
		},
		{
			name:       "strips query and fragment",
			raw:        pageURL + "?a=1#top",
			stripQuery: true,
			expected:   pageURL,
		},
		{
			name:       "normalizes numeric segment",
			raw:        "https://example.com/users/123",
			stripShape: true,
			expected:   "https://example.com/users/{}",
		},
		{
			name:       "numeric segments with different values normalize to same key",
			raw:        "https://example.com/users/456",
			stripShape: true,
			expected:   "https://example.com/users/{}",
		},
		{
			name:       "normalizes UUID segment",
			raw:        "https://example.com/items/550e8400-e29b-41d4-a716-446655440000",
			stripShape: true,
			expected:   "https://example.com/items/{}",
		},
		{
			name:       "normalizes date segment",
			raw:        "https://example.com/archive/2024-01-15",
			stripShape: true,
			expected:   "https://example.com/archive/{}",
		},
		{
			name:       "normalizes long hex segment",
			raw:        "https://example.com/t/abcdef0123456789abcdef",
			stripShape: true,
			expected:   "https://example.com/t/{}",
		},
		{
			name:       "leaves word segments alone",
			raw:        "https://example.com/about/team",
			stripShape: true,
			expected:   "https://example.com/about/team",
		},
		{
			name:       "leaves short hex-like word alone",
			raw:        "https://example.com/blog/feed",
			stripShape: true,
			expected:   "https://example.com/blog/feed",
		},
		{
			name:       "shapes path but keeps query",
			raw:        "https://example.com/users/123?a=1",
			stripShape: true,
			expected:   "https://example.com/users/{}?a=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := normalizeURL(tt.raw, tt.stripQuery, tt.stripShape); got != tt.expected {
				t.Errorf("normalizeURL(%q, %v, %v) = %q, want %q", tt.raw, tt.stripQuery, tt.stripShape, got, tt.expected)
			}
		})
	}
}
