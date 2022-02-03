package options

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

func (options *Options) Parse() (err error) {
	return
}
