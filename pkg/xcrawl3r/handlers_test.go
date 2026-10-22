//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"slices"
	"strings"
	"testing"
)

func TestMetaRefreshURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "plain refresh", content: "5; url=/next", want: testNextPath},
		{name: "quoted", content: `0; URL='/next'`, want: testNextPath},
		{name: "double quoted", content: `0; url="https://example.com/next"`, want: "https://example.com/next"},
		{name: "trailing text", content: "5; url=/next extra", want: testNextPath},
		{name: "missing", content: "5", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := metaRefreshURL(tt.content); got != tt.want {
				t.Fatalf("metaRefreshURL(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestSrcsetURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "descriptors", value: "a.jpg 1x, b.jpg 2x", want: []string{"a.jpg", "b.jpg"}},
		{name: "widths", value: "small.jpg 640w, large.jpg 1280w", want: []string{"small.jpg", "large.jpg"}},
		{name: "empty value", value: "", want: nil},
		{name: "blank entries", value: " , a.jpg ", want: []string{"a.jpg"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := srcsetURLs(tt.value); !slices.Equal(got, tt.want) {
				t.Fatalf("srcsetURLs(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestEventHandlerSelector(t *testing.T) {
	t.Parallel()

	selector := eventHandlerSelector()

	for _, attribute := range eventHandlerAttributes {
		if !strings.Contains(selector, "["+attribute+"]") {
			t.Fatalf("eventHandlerSelector() = %q, missing [%s]", selector, attribute)
		}
	}
}
