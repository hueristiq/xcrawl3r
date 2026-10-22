//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import "testing"

func TestDedupeKey(t *testing.T) {
	t.Parallel()

	t.Run("preserves scheme", func(t *testing.T) {
		t.Parallel()

		crawler := &Crawler{cfg: &Configuration{}}

		httpKey := crawler.dedupeKey("http://example.com/path")
		httpsKey := crawler.dedupeKey(testPathURL)

		if httpKey == httpsKey {
			t.Fatalf("dedupeKey() collapsed schemes: %q == %q", httpKey, httpsKey)
		}
	})

	t.Run("ignore query", func(t *testing.T) {
		t.Parallel()

		crawler := &Crawler{cfg: &Configuration{IgnoreQuery: true}}

		got := crawler.dedupeKey("https://example.com/path?b=2&a=1")
		want := testPathURL

		if got != want {
			t.Fatalf("dedupeKey() = %q, want %q", got, want)
		}
	})

	t.Run("shape dedup", func(t *testing.T) {
		t.Parallel()

		crawler := &Crawler{cfg: &Configuration{ShapeDedup: true}}

		got := crawler.dedupeKey("https://example.com/users/123/550e8400-e29b-41d4-a716-446655440000/2024-01/abcdef0123456789?q=1")
		want := "https://example.com/users/{}/{}/{}/{}?q=1"

		if got != want {
			t.Fatalf("dedupeKey() = %q, want %q", got, want)
		}
	})

	t.Run("trailing slash equivalence", func(t *testing.T) {
		t.Parallel()

		crawler := &Crawler{cfg: &Configuration{}}

		if got, want := crawler.dedupeKey("http://example.com"), crawler.dedupeKey("http://example.com/"); got != want {
			t.Fatalf("dedupeKey() = %q, want %q", got, want)
		}
	})
}
