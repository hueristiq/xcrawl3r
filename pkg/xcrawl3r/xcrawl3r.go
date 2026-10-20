package xcrawl3r

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

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

	headless *headless

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

	c.registerHandlers(ctx, collector, scope, results)

	for _, target := range c.targets(parsedTargetURL) {
		if ctx.Err() != nil {
			return
		}

		c.visit(ctx, target, results, scope, func() error {
			return collector.Visit(target)
		})
	}

	collector.Wait()
}

func (c *Crawler) targets(parsedTargetURL *hqgourlparser.URL) (targets []string) {
	targets = []string{parsedTargetURL.String()}

	if strings.Contains(parsedTargetURL.String(), ".min.") {
		targets = append(targets, strings.ReplaceAll(parsedTargetURL.String(), ".min.", "."))
	}

	robotsTXTURL := fmt.Sprintf("%s://%s/robots.txt", parsedTargetURL.Scheme, parsedTargetURL.Host)

	targets = append(targets, robotsTXTURL)

	for _, path := range sitemapPaths {
		sitemapURL := fmt.Sprintf("%s://%s%s", parsedTargetURL.Scheme, parsedTargetURL.Host, path)

		targets = append(targets, sitemapURL)
	}

	return
}

type ClientConfiguration struct {
	UserAgent string
}

type Configuration struct {
	Client             *ClientConfiguration
	Domains            []string
	IncludeSubdomains  bool
	Delay              time.Duration
	RandomDelay        time.Duration
	Timeout            time.Duration
	Retries            int
	RequestsPerMinute  int
	RandomUserAgent    bool
	InsecureSkipVerify bool
	Headers            []string
	Proxies            []string
	Depth              int
	Parallelism        int
	MaxPages           int
	CrawlDuration      time.Duration
	IgnoreQuery        bool
	ShapeDedup         bool
	Headless           bool
	HeadlessTimeout    time.Duration
}

func (cfg *Configuration) validate() (err error) {
	switch {
	case cfg.Timeout <= 0:
		return errors.New("timeout must be greater than 0")
	case cfg.Parallelism < 1:
		return errors.New("parallelism must be at least 1")
	case cfg.Depth < 0:
		return errors.New("depth must be 0 or greater")
	case cfg.MaxPages < 0:
		return errors.New("max pages must be 0 or greater")
	}

	return
}

var up = hqgourlparser.New(hqgourlparser.WithDefaultScheme("https"))

var sitemapPaths = []string{
	"/sitemap.xml",
	"/sitemap.xml.gz",
	"/sitemap_news.xml",
	"/sitemap_index.xml",
	"/sitemap-index.xml",
	"/sitemapindex.xml",
	"/sitemap-news.xml",
	"/wp-sitemap.xml",
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

	if cfg.Headless {
		timeout := cfg.HeadlessTimeout
		if timeout <= 0 {
			timeout = defaultHeadlessTimeout
		}

		crawler.headless = &headless{
			timeout: timeout,
		}
	}

	return
}
