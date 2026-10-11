package configuration_test

import (
	"strings"
	"testing"

	"github.com/hueristiq/xcrawl3r/internal/configuration"
)

func TestWithDefaultUserAgent(t *testing.T) {
	t.Parallel()

	t.Run("appends when absent", func(t *testing.T) {
		t.Parallel()

		headers := configuration.WithDefaultUserAgent([]string{"Accept: application/json"})

		if len(headers) != 2 {
			t.Fatalf("len(headers) = %d, want 2", len(headers))
		}

		if headers[1] != configuration.DefaultUserAgent {
			t.Errorf("headers[1] = %q, want %q", headers[1], configuration.DefaultUserAgent)
		}
	})

	t.Run("keeps configured user agent", func(t *testing.T) {
		t.Parallel()

		custom := "User-Agent: custom-agent/1.0"

		headers := configuration.WithDefaultUserAgent([]string{custom})

		if len(headers) != 1 {
			t.Fatalf("len(headers) = %d, want 1", len(headers))
		}

		if headers[0] != custom {
			t.Errorf("headers[0] = %q, want %q", headers[0], custom)
		}
	})

	t.Run("matches case-insensitively", func(t *testing.T) {
		t.Parallel()

		custom := "user-agent: custom-agent/1.0"

		headers := configuration.WithDefaultUserAgent([]string{"Accept: text/html", custom})

		if len(headers) != 2 {
			t.Fatalf("len(headers) = %d, want 2", len(headers))
		}

		for _, header := range headers {
			if strings.Contains(header, configuration.DefaultUserAgent) {
				t.Errorf("built-in user agent appended despite configured one: %q", headers)
			}
		}
	})
}
