package xcrawl3r

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

const (
	// maxSourceMapSources caps how many source entries of a source map are
	// processed.
	maxSourceMapSources = 10000

	// maxSourceMapContentSize caps how large a single sourcesContent entry
	// may be to be scanned.
	maxSourceMapContentSize = 2 << 20
)

// sourceMappingURLRegex extracts the sourceMappingURL directive browsers use
// to locate a script's source map.
var sourceMappingURLRegex = regexp.MustCompile(`(?i)sourceMappingURL\s*=\s*["']?([^"'\s]+)`)

type sourceMap struct {
	SourceRoot     string   `json:"sourceRoot"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent"`
}

// isSourceMapBody reports whether a response body looks like a source map.
func isSourceMapBody(rawPath, mediatype string, body []byte) (is bool) {
	if strings.HasSuffix(strings.ToLower(rawPath), ".map") {
		return true
	}

	return strings.HasSuffix(mediatype, "json") && bytes.Contains(body, []byte(`"sources"`))
}

// handleSourceMap extracts the source entries of a source map. Each source is
// resolved against the map's sourceRoot (itself resolved against mapURL) and
// treated as a link; each sourcesContent entry is scanned as JavaScript with
// its matching source URL as the base for relative references. mapURL is the
// URL the map was fetched from and visit queues a resolved URL.
func (c *Crawler) handleSourceMap(ctx context.Context, mapURL *url.URL, visit visitFunc, body []byte, scope *scope, results chan<- Result) {
	var sm sourceMap

	if err := json.Unmarshal(body, &sm); err != nil {
		return
	}

	if len(sm.Sources) > maxSourceMapSources {
		sm.Sources = sm.Sources[:maxSourceMapSources]
	}

	sourceBase := mapURL

	if sm.SourceRoot != "" {
		sourceBase = resolveSourceRoot(mapURL, sm.SourceRoot)
	}

	for _, source := range sm.Sources {
		reference := resolveReference(sourceBase, source)
		if reference == "" {
			continue
		}

		c.processLinkFrom(ctx, mapURL, visit, reference, "sourcemap", scope, results)
	}

	for index, content := range sm.SourcesContent {
		if index >= len(sm.Sources) || len(content) > maxSourceMapContentSize {
			continue
		}

		reference := resolveReference(sourceBase, sm.Sources[index])

		sourceURL, err := url.Parse(reference)
		if err != nil || sourceURL.String() == "" {
			sourceURL = mapURL
		}

		c.handleJSSnippetFrom(ctx, sourceURL, visit, content, "sourcemap:content", scope, results)
	}
}

// resolveSourceRoot resolves sourceRoot as a directory URL. Relative roots
// are resolved against the map URL; a trailing slash is added so that
// ResolveReference treats the root as a directory rather than a file.
func resolveSourceRoot(mapURL *url.URL, sourceRoot string) (base *url.URL) {
	root, err := url.Parse(sourceRoot)
	if err != nil {
		return mapURL
	}

	if !strings.HasSuffix(root.Path, "/") {
		root.Path += "/"
		root.RawPath = ""
	}

	if root.IsAbs() {
		return root
	}

	return mapURL.ResolveReference(root)
}
