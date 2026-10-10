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
	opts *options

	urlExtractorRegex *regexp.Regexp

	fileURLsToRequestExtRegex    *regexp.Regexp
	fileURLsNotToRequestExtRegex *regexp.Regexp

	storage *storage.InMemoryStorage

	headless *headless

	seen sync.Map
}

func New(ofs ...OptionFunc) (crawler *Crawler, err error) {
	crawler = &Crawler{
		opts: defaultOptions(),
	}

	for _, f := range ofs {
		f(crawler.opts)
	}

	crawler.urlExtractorRegex = hqgourlextractor.New().CompileRegex()

	crawler.fileURLsToRequestExtRegex = regexp.MustCompile(`\.(css|csv|js|json|map|txt|xml|yaml|yml)$`)
	crawler.fileURLsNotToRequestExtRegex = regexp.MustCompile(`\.(apng|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf)$`)

	crawler.storage = &storage.InMemoryStorage{}

	if crawler.opts.headless {
		timeout := crawler.opts.headlessTimeout
		if timeout <= 0 {
			timeout = defaultHeadlessTimeout
		}

		crawler.headless = &headless{
			timeout: timeout,
		}
	}

	return
}

func (c *Crawler) Crawl(ctx context.Context, target string) <-chan Result {
	results := make(chan Result)

	go c.crawl(ctx, target, results)

	return results
}

func (c *Crawler) crawl(ctx context.Context, target string, results chan Result) {
	defer close(results)

	if c.opts.crawlDuration > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, c.opts.crawlDuration)
		defer cancel()
	}

	parsedTargetURL, err := up.Parse(target)
	if err != nil {
		emit(ctx, results, Result{
			Type:  ResultError,
			Error: fmt.Errorf("error parsing target %s: %w", target, err),
		})

		return
	}

	scope := c.scopeFor(parsedTargetURL)

	collector, err := c.collector(ctx, scope)
	if err != nil {
		emit(ctx, results, Result{
			Type:  ResultError,
			Error: fmt.Errorf("error creating collector for %s: %w", target, err),
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

var up = hqgourlparser.New(hqgourlparser.WithDefaultScheme("https"))
