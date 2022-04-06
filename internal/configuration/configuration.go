package configuration

import (
	"github.com/logrusorgru/aurora/v3"
)

const (
	DefaultConcurrency    = 5
	DefaultDepth          = 1
	DefaultThreads        = 20
	DefaultMaxRandomDelay = 60
	DefaultTimeout        = 10
	VERSION               = "v1.0.0"
)

var (
	BANNER string = aurora.Sprintf(aurora.BrightBlue(`
     _                          _ _____      
 ___(_) __ _ _ __ __ ___      _| |___ / _ __ 
/ __| |/ _`+"`"+` | '__/ _`+"`"+` \ \ /\ / / | |_ \| '__|
\__ \ | (_| | | | (_| |\ V  V /| |___) | |   
|___/_|\__, |_|  \__,_| \_/\_/ |_|____/|_| %s
       |___/
`).Bold(), aurora.BrightRed(VERSION).Bold())
)

type Configuration struct {
	AllowedDomains    []string
	Concurrency       int
	Debug             bool
	Delay             int
	Depth             int
	Headless          bool
	IncludeSubdomains bool
	MaxRandomDelay    int // seconds
	Proxy             string
	Render            bool
	RenderTimeout     int // seconds
	Threads           int
	Timeout           int // seconds
	UserAgent         string
}

func (configuration *Configuration) Validate() (err error) {
	if configuration.Concurrency <= 0 {
		configuration.Concurrency = DefaultConcurrency
	}

	if configuration.Depth <= 0 {
		configuration.Depth = DefaultDepth
	}

	if configuration.Threads <= 0 {
		configuration.Threads = DefaultThreads
	}

	if configuration.Timeout <= 0 {
		configuration.Timeout = DefaultTimeout
	}

	return
}
