//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"reflect"
	"testing"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func TestRenderOptions(t *testing.T) {
	t.Parallel()

	defaults := len(chromedp.DefaultExecAllocatorOptions)

	tests := []struct {
		name string
		cfg  *Configuration
		want int
	}{
		{name: "nil", cfg: nil, want: defaults},
		{name: "empty config", cfg: &Configuration{}, want: defaults},
		{name: "proxy", cfg: &Configuration{Proxies: []string{"http://127.0.0.1:8080"}}, want: defaults + 1},
		{name: "insecure", cfg: &Configuration{InsecureSkipVerify: true}, want: defaults + 1},
		{name: "user agent", cfg: &Configuration{Client: &ClientConfiguration{UserAgent: "test"}}, want: defaults + 1},
		{
			name: "all options",
			cfg: &Configuration{
				Proxies:            []string{"http://127.0.0.1:8080"},
				InsecureSkipVerify: true,
				Client:             &ClientConfiguration{UserAgent: "test"},
			},
			want: defaults + 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := len(renderOptions(tt.cfg)); got != tt.want {
				t.Fatalf("len(renderOptions()) = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCookieParams(t *testing.T) {
	t.Parallel()

	const rawURL = "https://example.com/page"

	got := cookieParams([]string{
		"session=abc",
		"token=def; Path=/; Secure",
		" spaced = value ",
		"invalid",
		"=value",
		"",
	}, rawURL)

	want := []network.SetCookieParams{
		{Name: "session", Value: "abc", URL: rawURL},
		{Name: "token", Value: "def", URL: rawURL},
		{Name: "spaced", Value: "value", URL: rawURL},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cookieParams() = %#v, want %#v", got, want)
	}
}

func TestParseHeaderEntries(t *testing.T) {
	t.Parallel()

	got := parseHeaderEntries([]string{
		"Content-Type: text/html",
		"X-Test: a:b",
		"NoColon",
		": empty",
		" A : b ",
	})

	want := network.Headers{
		"Content-Type": mediaTypeHTML,
		"X-Test":       "a:b",
		"A":            "b",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseHeaderEntries() = %#v, want %#v", got, want)
	}
}
