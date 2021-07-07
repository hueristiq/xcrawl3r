package sigrawl3r

type Options struct {
	Debug       bool
	Depth       int
	IncludeSubs bool
	Proxies     string
	RandomDelay int
	Threads     int
	Timeout     int
	UserAgent   string
}

func ParseOptions(options *Options) (*Options, error) {
	return options, nil
}
