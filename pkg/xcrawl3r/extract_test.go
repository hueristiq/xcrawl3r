//nolint:testpackage // white-box tests for unexported helpers.
package xcrawl3r

import (
	"strings"
	"testing"
)

func TestIsExtractableMediaType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mediatype string
		want      bool
	}{
		{mediatype: mediaTypeHTML, want: true},
		{mediatype: testMediaPlain, want: true},
		{mediatype: mediaTypeJSON, want: true},
		{mediatype: mediaTypeJavaScript, want: true},
		{mediatype: "application/xml", want: true},
		{mediatype: mediaTypeXHTML, want: true},
		{mediatype: "application/ld+json", want: true},
		{mediatype: "application/atom+xml", want: true},
		{mediatype: testMediaBinary, want: false},
		{mediatype: "image/png", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.mediatype, func(t *testing.T) {
			t.Parallel()

			if got := isExtractableMediaType(tt.mediatype); got != tt.want {
				t.Fatalf("isExtractableMediaType(%q) = %v, want %v", tt.mediatype, got, tt.want)
			}
		})
	}
}

func TestIsExtractableRelative(t *testing.T) {
	t.Parallel()

	cssBody := "a{background:url(/img.png)}"

	cssIndex := strings.Index(cssBody, "/img.png")
	if cssIndex < 0 {
		t.Fatal("fixture lost /img.png")
	}

	if !isExtractableRelative(cssBody, []int{cssIndex, cssIndex + len("/img.png")}) {
		t.Fatal("isExtractableRelative(css url) = false, want true")
	}

	htmlBody := "</div>"

	htmlIndex := strings.Index(htmlBody, "/div")
	if htmlIndex < 0 {
		t.Fatal("fixture lost /div")
	}

	if isExtractableRelative(htmlBody, []int{htmlIndex, htmlIndex + len("/div")}) {
		t.Fatal("isExtractableRelative(html tag tail) = true, want false")
	}
}

func TestIsJSBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mediatype string
		rawPath   string
		want      bool
	}{
		{name: "javascript type", mediatype: mediaTypeJavaScript, rawPath: "/app", want: true},
		{name: "json type", mediatype: mediaTypeJSON, rawPath: testDataPath, want: true},
		{name: "js extension", mediatype: testMediaPlain, rawPath: "/app.js", want: true},
		{name: "mjs extension", mediatype: testMediaPlain, rawPath: "/app.mjs", want: true},
		{name: "html document", mediatype: mediaTypeHTML, rawPath: testIndexPath, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isJSBody(tt.mediatype, tt.rawPath); got != tt.want {
				t.Fatalf("isJSBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsJSCSSBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mediatype string
		rawPath   string
		want      bool
	}{
		{name: "javascript type", mediatype: mediaTypeJavaScript, rawPath: "/app", want: true},
		{name: "javascript extension", mediatype: testMediaPlain, rawPath: "/app.js", want: true},
		{name: "json type", mediatype: mediaTypeJSON, rawPath: testDataPath, want: true},
		{name: "map extension", mediatype: testMediaBinary, rawPath: "/app.js.map", want: true},
		{name: "css type", mediatype: "text/css", rawPath: "/style", want: true},
		{name: "css extension", mediatype: testMediaPlain, rawPath: "/style.css", want: true},
		{name: "html document", mediatype: mediaTypeHTML, rawPath: testIndexPath, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isJSCSSBody(tt.mediatype, tt.rawPath); got != tt.want {
				t.Fatalf("isJSCSSBody() = %v, want %v", got, tt.want)
			}
		})
	}
}
