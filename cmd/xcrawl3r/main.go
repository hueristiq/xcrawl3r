package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hueristiq/xcrawl3r/internal/configuration"
	"github.com/hueristiq/xcrawl3r/pkg/stdio"
	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
	"github.com/logrusorgru/aurora/v4"
	"github.com/spf13/pflag"
	"go.source.hueristiq.com/logger"
	"go.source.hueristiq.com/logger/formatter"
	"go.source.hueristiq.com/logger/levels"
)

var (
	au = aurora.New(aurora.WithColors(true))

	URLs         []string
	URLsFilePath string

	domain            string
	includeSubdomains bool

	depth     int
	headless  bool
	headers   []string
	proxies   []string
	render    bool
	timeout   int
	userAgent string

	concurrency    int
	delay          int
	maxRandomDelay int
	parallelism    int

	debug      bool
	monochrome bool
	output     string

	silent  bool
	verbose bool
)

func init() {
	// Handle command line arguments & flags
	pflag.StringSliceVarP(&URLs, "url", "u", []string{}, "")
	pflag.StringVarP(&URLsFilePath, "list", "l", "", "")

	pflag.StringVarP(&domain, "domain", "d", "", "")
	pflag.BoolVar(&includeSubdomains, "include-subdomains", false, "")

	pflag.IntVar(&depth, "depth", 3, "")
	pflag.BoolVar(&headless, "headless", false, "")
	pflag.StringSliceVarP(&headers, "headers", "H", []string{}, "")
	pflag.StringSliceVar(&proxies, "proxy", []string{}, "")
	pflag.BoolVar(&render, "render", false, "")
	pflag.IntVar(&timeout, "timeout", 10, "")
	pflag.StringVar(&userAgent, "user-agent", xcrawl3r.DefaultUserAgent, "")

	pflag.IntVarP(&concurrency, "concurrency", "c", 10, "")
	pflag.IntVar(&delay, "delay", 0, "")
	pflag.IntVar(&maxRandomDelay, "max-random-delay", 1, "")
	pflag.IntVarP(&parallelism, "parallelism", "p", 10, "")

	pflag.BoolVar(&debug, "debug", false, "")
	pflag.BoolVarP(&monochrome, "monochrome", "m", false, "")
	pflag.StringVarP(&output, "output", "o", "", "")

	pflag.BoolVar(&silent, "silent", false, "")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "")

	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		logger.Info().Label("").Msg(configuration.BANNER(au))

		h := "USAGE:\n"
		h += fmt.Sprintf(" %s [OPTIONS]\n", configuration.NAME)

		h += "\nINPUT:\n"
		h += " -u, --url string[]             target URL\n"
		h += " -l, --list string                 target URLs list file path\n"

		h += "\nSCOPE:\n"
		h += " -d, --domain string               domain to match URLs\n"
		h += "     --include-subdomains bool     match subdomains' URLs\n"

		h += "\nCONFIGURATION:\n"
		h += "     --depth int                   maximum depth to crawl (default 3)\n"
		h += "                                      TIP: set it to `0` for infinite recursion\n"
		h += "     --headless bool               If true the browser will be displayed while crawling.\n"
		h += " -H, --headers string[]            custom header to include in requests\n"
		h += "                                      e.g. -H 'Referer: http://example.com/'\n"
		h += "                                      TIP: use multiple flag to set multiple headers\n"
		h += "     --proxy string[]              Proxy URL (e.g: http://127.0.0.1:8080)\n"
		h += "                                      TIP: use multiple flag to set multiple proxies\n"
		h += "     --render bool                 utilize a headless chrome instance to render pages\n"
		h += "     --timeout int                 time to wait for request in seconds (default: 10)\n"
		h += fmt.Sprintf("     --user-agent string           User Agent to use (default: %s)\n", xcrawl3r.DefaultUserAgent)
		h += "                                      TIP: use `web` for a random web user-agent,\n"
		h += "                                      `mobile` for a random mobile user-agent,\n"
		h += "                                       or you can set your specific user-agent.\n"

		h += "\nRATE LIMIT:\n"
		h += " -c, --concurrency int             number of concurrent fetchers to use (default 10)\n"
		h += "     --delay int                   delay between each request in seconds\n"
		h += "     --max-random-delay int        maximux extra randomized delay added to `--dalay` (default: 1s)\n"
		h += " -p, --parallelism int             number of concurrent URLs to process (default: 10)\n"

		h += "\nOUTPUT:\n"
		h += "     --debug bool                  enable debug mode (default: false)\n"
		h += " -m, --monochrome bool             coloring: no colored output mode\n"
		h += " -o, --output string               output file to write found URLs\n"
		h += "     --silent bool                 display output URLs only\n"
		h += " -v, --verbose bool                display verbose output\n"

		logger.Info().Label("").Msg(h)
		logger.Print().Msg("")
	}

	pflag.Parse()

	logger.DefaultLogger.SetFormatter(formatter.NewConsoleFormatter(&formatter.ConsoleFormatterConfiguration{
		Colorize: !monochrome,
	}))

	if verbose {
		logger.DefaultLogger.SetMaxLogLevel(levels.LevelDebug)
	}

	if silent {
		logger.DefaultLogger.SetMaxLogLevel(levels.LevelSilent)
	}

	au = aurora.New(aurora.WithColors(!monochrome))
}

func main() {
	logger.Info().Label("").Msg(configuration.BANNER(au))

	var err error

	// load input URLs from file
	if URLsFilePath != "" {
		var file *os.File

		file, err = os.Open(URLsFilePath)
		if err != nil {
			logger.Error().Msg(err.Error())
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			URL := scanner.Text()

			if URL != "" {
				URLs = append(URLs, URL)
			}
		}

		if err = scanner.Err(); err != nil {
			logger.Error().Msg(err.Error())
		}
	}

	// load input URLs from stdin
	if stdio.HasStdIn() {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			URL := scanner.Text()

			if URL != "" {
				URLs = append(URLs, URL)
			}
		}

		if err = scanner.Err(); err != nil {
			logger.Error().Msg(err.Error())
		}
	}

	logger.Info().Msgf("Crawling URLs for %s...", au.Underline(len(URLs)).Bold())
	logger.Print().Msg("")

	cfg := &xcrawl3r.Configuration{
		URLs: URLs,

		Domain:            domain,
		IncludeSubdomains: includeSubdomains,

		Depth:     depth,
		Headless:  headless,
		Headers:   headers,
		Proxies:   proxies,
		Render:    render,
		Timeout:   timeout,
		UserAgent: userAgent,

		Concurrency:    concurrency,
		Delay:          delay,
		MaxRandomDelay: maxRandomDelay,
		Parallelism:    parallelism,

		Debug: debug,
	}

	crawler, err := xcrawl3r.New(cfg)
	if err != nil {
		logger.Fatal().Msg(err.Error())
	}

	var writer *bufio.Writer

	if output != "" {
		directory := filepath.Dir(output)

		if _, err := os.Stat(directory); os.IsNotExist(err) {
			if err = os.MkdirAll(directory, os.ModePerm); err != nil {
				logger.Fatal().Msg(err.Error())
			}
		}

		var file *os.File

		file, err = os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			logger.Fatal().Msg(err.Error())
		}

		defer file.Close()

		writer = bufio.NewWriter(file)
	}

	for URL := range crawler.Crawl() {
		switch URL.Type {
		case xcrawl3r.ResultError:
			if verbose {
				logger.Error().Msgf("%s: %s\n", URL.Source, URL.Error)
			}
		case xcrawl3r.ResultURL:
			if verbose {
				logger.Print().Msgf("[%s] %s", au.BrightBlue(URL.Source), URL.Value)
			} else {
				logger.Print().Msg(URL.Value)
			}

			if writer != nil {
				fmt.Fprintln(writer, URL.Value)

				if err := writer.Flush(); err != nil {
					logger.Fatal().Msg(err.Error())
				}
			}
		}
	}
}
