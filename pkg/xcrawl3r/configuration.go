package xcrawl3r

import (
	"errors"
	"time"
)

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
	Cookies            []string
	Proxies            []string
	Depth              int
	Parallelism        int
	MaxPages           int
	CrawlDuration      time.Duration
	IgnoreQuery        bool
	ShapeDedup         bool
	Render             bool
	RenderTimeout      time.Duration
	RenderSettle       time.Duration
}

func (cfg *Configuration) validate() (err error) {
	switch {
	case cfg.Timeout <= 0:
		return errors.New("timeout must be greater than 0")
	case cfg.Parallelism < 1:
		return errors.New("parallelism must be at least 1")
	case cfg.Delay < 0:
		return errors.New("delay must be 0 or greater")
	case cfg.RandomDelay < 0:
		return errors.New("random delay must be 0 or greater")
	case cfg.CrawlDuration < 0:
		return errors.New("crawl duration must be 0 or greater")
	case cfg.Depth < 0:
		return errors.New("depth must be 0 or greater")
	case cfg.MaxPages < 0:
		return errors.New("max pages must be 0 or greater")
	}

	return
}
