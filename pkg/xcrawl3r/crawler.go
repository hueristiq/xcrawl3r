package xcrawl3r

import (
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

	seen sync.Map
}

func (c *Crawler) Crawl(target string) <-chan Result {
	results := make(chan Result)

	go func() {
		defer close(results)

		parsedTargetURL, err := up.Parse(target)
		if err != nil {
			results <- Result{
				Type:  ResultError,
				Error: fmt.Errorf("error parsing target %s: %w", target, err),
			}

			return
		}

		scope := c.scopeFor(parsedTargetURL)

		collector, err := c.collector(scope)
		if err != nil {
			results <- Result{
				Type:  ResultError,
				Error: fmt.Errorf("error creating collector for %s: %w", target, err),
			}

			return
		}

		c.registerHandlers(collector, scope, results)

		for _, target := range c.targets(parsedTargetURL) {
			c.visit(target, results, func() error {
				return collector.Visit(target)
			})
		}

		collector.Wait()
	}()

	return results
}

type Configuration struct {
	Domains           []string
	IncludeSubdomains bool
	Delay             int
	RandomDelay       int
	Timeout           int
	Retries           int
	RequestsPerMinute int
	Headers           []string
	Proxies           []string
	Depth             int
	Parallelism       int
	Debug             bool
}

const defaultRetries = 3

var up = hqgourlparser.New(hqgourlparser.WithDefaultScheme("https"))

func New(cfg *Configuration) (crawler *Crawler, err error) {
	crawler = &Crawler{
		cfg: cfg,
	}

	if cfg.Retries == 0 {
		cfg.Retries = defaultRetries
	}

	crawler.urlExtractorRegex = hqgourlextractor.New().CompileRegex()

	crawler.fileURLsToRequestExtRegex = regexp.MustCompile(`\.(css|csv|js|json|map|txt|xml|yaml|yml)$`)
	crawler.fileURLsNotToRequestExtRegex = regexp.MustCompile(`\.(apng|png|bmp|gif|heif|ico|cur|jpg|jpeg|jfif|pjp|pjpeg|psd|raw|svg|tif|tiff|webp|xbm|3gp|aac|flac|mpg|mpeg|mp3|mp4|m4a|m4v|m4p|oga|ogg|ogv|mov|wav|webm|eot|woff|woff2|ttf|otf)$`)

	crawler.storage = &storage.InMemoryStorage{}

	return
}
