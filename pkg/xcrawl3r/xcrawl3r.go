package xcrawl3r

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/gocolly/colly/v2/storage"
	hqgourlextractor "github.com/hueristiq/hq-lib-url-go/extractor"
	hqgourlparser "github.com/hueristiq/hq-lib-url-go/parser"
)

type Crawler struct {
	cfg *Configuration

	urlExtractorRegex *regexp.Regexp

	fileURLsToRequestExtRegex    *regexp.Regexp
	fileURLsNotToRequestExtRegex *regexp.Regexp

	storage *storage.InMemoryStorage

	renderer *renderer

	seen sync.Map
}

func (c *Crawler) Crawl(ctx context.Context, target string) (results chan Result) {
	results = make(chan Result)

	go c.crawl(ctx, target, results)

	return
}

func (c *Crawler) crawl(ctx context.Context, target string, results chan Result) {
	defer close(results)

	if c.cfg.CrawlDuration > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, c.cfg.CrawlDuration)
		defer cancel()
	}

	parsedTargetURL, err := up.Parse(target)
	if err != nil {
		emit(ctx, results, Result{
			Type:   ResultError,
			Source: "xcrawl3r",
			Error:  fmt.Errorf("failed to parse target %q: %w", target, err),
		})

		return
	}

	scope := c.scopeFor(parsedTargetURL)

	collector, err := c.collector(ctx, scope, results)
	if err != nil {
		emit(ctx, results, Result{
			Type:   ResultError,
			Source: "xcrawl3r",
			Error:  fmt.Errorf("failed to create collector for %q: %w", target, err),
		})

		return
	}

	// Registered after close(results) so it runs before the channel closes on
	// every return path, including cancellation.
	defer collector.Wait()

	c.registerHandlers(ctx, collector, scope, results)

	for _, target := range c.targets(parsedTargetURL, scope) {
		if ctx.Err() != nil {
			return
		}

		c.visit(ctx, target, results, func() error {
			return collector.Visit(target)
		})
	}
}

func (c *Crawler) targets(parsedTargetURL *hqgourlparser.URL, scope *scope) (targets []string) {
	targets = []string{parsedTargetURL.String()}

	if companion, ok := minifiedCompanion(parsedTargetURL.String()); ok && scope.allows(companion) {
		targets = append(targets, companion)
	}

	robotsTXTURL := fmt.Sprintf("%s://%s/robots.txt", parsedTargetURL.Scheme, parsedTargetURL.Host)

	targets = append(targets, robotsTXTURL)

	for _, path := range sitemapPaths {
		sitemapURL := fmt.Sprintf("%s://%s%s", parsedTargetURL.Scheme, parsedTargetURL.Host, path)

		targets = append(targets, sitemapURL)
	}

	return
}

// New returns a crawler configured with cfg. Every crawler owns fresh
// internal state, so crawlers are safe to construct and use concurrently.
//
// Parameters:
//   - cfg (*Configuration): the crawler configuration.
//
// Returns:
//   - crawler (*Crawler): the configured crawler.
//   - err (error): non-nil when the configuration is invalid or the
//     crawler's dependencies cannot be initialized.
func New(cfg *Configuration) (crawler *Crawler, err error) {
	crawler = &Crawler{
		cfg: cfg,
	}

	if err = cfg.validate(); err != nil {
		return
	}

	if crawler.urlExtractorRegex, err = hqgourlextractor.New().CompileRegex(); err != nil {
		err = fmt.Errorf("failed to compile URL extractor regex: %w", err)

		return
	}

	crawler.fileURLsToRequestExtRegex = regexp.MustCompile(`\.(css|csv|js|json|map|txt|xml|yaml|yml)$`)
	crawler.fileURLsNotToRequestExtRegex = regexp.MustCompile(`\.(apng|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf)$`)

	crawler.storage = &storage.InMemoryStorage{}

	if err = crawler.storage.Init(); err != nil {
		err = fmt.Errorf("failed to initialize storage: %w", err)

		return
	}

	if cfg.Render {
		timeout := cfg.RenderTimeout
		if timeout <= 0 {
			timeout = defaultRenderTimeout
		}

		settle := cfg.RenderSettle
		if settle <= 0 {
			settle = defaultRenderSettle
		}

		crawler.renderer = &renderer{
			cfg:     cfg,
			timeout: timeout,
			settle:  settle,
		}
	}

	return
}
