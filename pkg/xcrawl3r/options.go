package xcrawl3r

import "time"

type OptionFunc func(*options)

type options struct {
	domains           []string
	includeSubdomains bool
	delay             time.Duration
	randomDelay       time.Duration
	timeout           time.Duration
	retries           int
	requestsPerMinute int
	ignoreQuery       bool
	shapeDedup        bool
	randomUserAgent   bool
	headers           []string
	proxies           []string
	depth             int
	parallelism       int
	maxPages          int
	crawlDuration     time.Duration
	headless          bool
	headlessTimeout   time.Duration
	debug             bool
}

const (
	defaultRetries     = 3
	defaultTimeout     = 10 * time.Second
	defaultDepth       = 1
	defaultParallelism = 5
)

func defaultOptions() (opts *options) {
	return &options{
		timeout:     defaultTimeout,
		retries:     defaultRetries,
		depth:       defaultDepth,
		parallelism: defaultParallelism,
	}
}

func WithDomains(domains []string) (f OptionFunc) {
	return func(opts *options) {
		opts.domains = domains
	}
}

func WithIncludeSubdomains(includeSubdomains bool) (f OptionFunc) {
	return func(opts *options) {
		opts.includeSubdomains = includeSubdomains
	}
}

func WithDelay(delay time.Duration) (f OptionFunc) {
	return func(opts *options) {
		opts.delay = delay
	}
}

func WithRandomDelay(randomDelay time.Duration) (f OptionFunc) {
	return func(opts *options) {
		opts.randomDelay = randomDelay
	}
}

func WithTimeout(timeout time.Duration) (f OptionFunc) {
	return func(opts *options) {
		opts.timeout = timeout
	}
}

func WithRetries(retries int) (f OptionFunc) {
	return func(opts *options) {
		switch {
		case retries < 0:
			opts.retries = 1
		case retries == 0:
			opts.retries = defaultRetries
		default:
			opts.retries = retries
		}
	}
}

func WithRequestsPerMinute(requestsPerMinute int) (f OptionFunc) {
	return func(opts *options) {
		opts.requestsPerMinute = requestsPerMinute
	}
}

func WithIgnoreQuery(ignoreQuery bool) (f OptionFunc) {
	return func(opts *options) {
		opts.ignoreQuery = ignoreQuery
	}
}

func WithShapeDedup(shapeDedup bool) (f OptionFunc) {
	return func(opts *options) {
		opts.shapeDedup = shapeDedup
	}
}

func WithRandomUserAgent(randomUserAgent bool) (f OptionFunc) {
	return func(opts *options) {
		opts.randomUserAgent = randomUserAgent
	}
}

func WithMaxPages(maxPages int) (f OptionFunc) {
	return func(opts *options) {
		opts.maxPages = maxPages
	}
}

func WithCrawlDuration(crawlDuration time.Duration) (f OptionFunc) {
	return func(opts *options) {
		opts.crawlDuration = crawlDuration
	}
}

func WithHeadless(headless bool) (f OptionFunc) {
	return func(opts *options) {
		opts.headless = headless
	}
}

func WithHeadlessTimeout(headlessTimeout time.Duration) (f OptionFunc) {
	return func(opts *options) {
		opts.headlessTimeout = headlessTimeout
	}
}

func WithHeaders(headers ...string) (f OptionFunc) {
	return func(opts *options) {
		opts.headers = headers
	}
}

func WithProxies(proxies ...string) (f OptionFunc) {
	return func(opts *options) {
		opts.proxies = proxies
	}
}

func WithDepth(depth int) (f OptionFunc) {
	return func(opts *options) {
		opts.depth = depth
	}
}

func WithParallelism(parallelism int) (f OptionFunc) {
	return func(opts *options) {
		opts.parallelism = parallelism
	}
}

func WithDebug(debug bool) (f OptionFunc) {
	return func(opts *options) {
		opts.debug = debug
	}
}
