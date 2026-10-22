package xcrawl3r

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/gocolly/colly/v2"
)

// visitFunc fetches a discovered URL. colly.Request.Visit satisfies it.
type visitFunc func(rawURL string) error

// processLink resolves link against request's URL, and when the result is in
// scope and new, emits it as a result and queues a visit.
func (c *Crawler) processLink(ctx context.Context, request *colly.Request, link, via string, scope *scope, results chan<- Result) {
	c.processLinkFrom(ctx, request.URL, request.Visit, link, via, scope, results)
}

// processLinkFrom is processLink with an explicit base URL and visit function,
// used when a link belongs to a URL other than the one that was requested.
func (c *Crawler) processLinkFrom(ctx context.Context, baseURL *url.URL, visit visitFunc, link, via string, scope *scope, results chan<- Result) {
	c.handleLinkFrom(ctx, baseURL, visit, link, via, scope, results, true)
}

// emitLinkFrom records link as a result without queueing a visit. It is used
// for links containing EXPR placeholders that cannot be fetched as they are.
func (c *Crawler) emitLinkFrom(ctx context.Context, baseURL *url.URL, link, via string, scope *scope, results chan<- Result) {
	c.handleLinkFrom(ctx, baseURL, nil, link, via, scope, results, false)
}

func (c *Crawler) handleLinkFrom(ctx context.Context, baseURL *url.URL, visit visitFunc, link, via string, scope *scope, results chan<- Result, shouldVisit bool) {
	absoluteURL := resolveReference(baseURL, link)
	if absoluteURL == "" {
		return
	}

	canonicalURL := normalizeURL(absoluteURL)

	if !scope.allows(canonicalURL) || c.alreadySeen(canonicalURL) {
		return
	}

	result := Result{
		Type:   ResultURL,
		Source: baseURL.String(),
		Value:  canonicalURL,
		Via:    via,
	}

	if !emit(ctx, results, result) {
		return
	}

	if !shouldVisit || visit == nil {
		return
	}

	c.visit(ctx, canonicalURL, results, func() error {
		return visit(canonicalURL)
	})
}

func (c *Crawler) visit(ctx context.Context, rawURL string, results chan<- Result, do func() error) {
	err := do()
	if err == nil {
		return
	}

	if !isExpectedVisitError(err) {
		emit(ctx, results, Result{
			Type:  ResultError,
			Error: fmt.Errorf("failed to visit %s: %w", rawURL, err),
		})
	}
}

func (c *Crawler) alreadySeen(rawURL string) (seen bool) {
	_, seen = c.seen.LoadOrStore(c.dedupeKey(rawURL), struct{}{})

	return
}

func emit(ctx context.Context, results chan<- Result, result Result) (emitted bool) {
	select {
	case results <- result:
		return true
	case <-ctx.Done():
		return false
	}
}

func isExpectedVisitError(err error) (expected bool) {
	var alreadyVisitedError *colly.AlreadyVisitedError

	switch {
	case errors.As(err, &alreadyVisitedError),
		errors.Is(err, colly.ErrMaxDepth),
		errors.Is(err, colly.ErrNoURLFiltersMatch),
		errors.Is(err, colly.ErrForbiddenDomain),
		errors.Is(err, colly.ErrRobotsTxtBlocked):
		return true
	}

	return
}
