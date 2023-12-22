package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hueristiq/hqgolog"
	"github.com/hueristiq/hqgolog/formatter"
	"github.com/hueristiq/hqgolog/levels"
	"github.com/hueristiq/hqgourl"
	"github.com/hueristiq/xcrawl3r/internal/configuration"
	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
	"github.com/logrusorgru/aurora/v3"
	"github.com/spf13/pflag"
)

var (
	au aurora.Aurora

	domain            string
	includeSubdomains bool
	seedsFile         string
	URL               string

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
	verbosity  string
)

func init() {
	// Handle command line arguments & flags
	pflag.StringVarP(&domain, "domain", "d", "", "")
	pflag.BoolVar(&includeSubdomains, "include-subdomains", false, "")
	pflag.StringVarP(&seedsFile, "seeds", "s", "", "")
	pflag.StringVarP(&URL, "url", "u", "", "")

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
	pflag.StringVarP(&verbosity, "verbosity", "v", string(levels.LevelInfo), "")

	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		fmt.Fprintln(os.Stderr, configuration.BANNER)

		h := "\nUSAGE:\n"
		h += "  xcrawl3r [OPTIONS]\n"

		h += "\nINPUT:\n"
		h += " -d, --domain string               domain to match URLs\n"
		h += "     --include-subdomains bool     match subdomains' URLs\n"
		h += " -s, --seeds string                seed URLs file (use `-` to get from stdin)\n"
		h += " -u, --url string                  URL to crawl\n"

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
		h += " -v, --verbosity string            debug, info, warning, error, fatal or silent (default: debug)\n"

		fmt.Fprint(os.Stderr, h)
	}

	pflag.Parse()

	// Initialize logger
	hqgolog.DefaultLogger.SetMaxLevel(levels.LevelStr(verbosity))
	hqgolog.DefaultLogger.SetFormatter(formatter.NewCLI(&formatter.CLIOptions{
		Colorize: !monochrome,
	}))

	au = aurora.NewAurora(!monochrome)
}

func main() {
	if verbosity != string(levels.LevelSilent) {
		fmt.Fprintln(os.Stderr, configuration.BANNER)
	}

	if seedsFile != "" && URL == "" && domain == "" {
		hqgolog.Fatal().Msg("using `-s, --seeds` requires either `-d, --domain` or `-u, --url` to be set!")
	}

	// Load input URLs
	seeds := []string{}

	if URL != "" {
		seeds = append(seeds, URL)

		if domain == "" {
			domain = URL
		}
	}

	if seedsFile != "" {
		var (
			err  error
			file *os.File
			stat fs.FileInfo
		)

		switch {
		case seedsFile != "" && seedsFile == "-":
			stat, err = os.Stdin.Stat()
			if err != nil {
				hqgolog.Fatal().Msg("no stdin")
			}

			if stat.Mode()&os.ModeNamedPipe == 0 {
				hqgolog.Fatal().Msg("no stdin")
			}

			file = os.Stdin
		case seedsFile != "" && seedsFile != "-":
			file, err = os.Open(seedsFile)
			if err != nil {
				hqgolog.Fatal().Msg(err.Error())
			}
		default:
			hqgolog.Fatal().Msg("xcrawl3r takes input from stdin or file using a flag")
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			inputURL := scanner.Text()

			if inputURL != "" {
				seeds = append(seeds, inputURL)
			}
		}

		if scanner.Err() != nil {
			hqgolog.Fatal().Msgf("%s", err)
		}
	}

	parsedURL, err := hqgourl.Parse(domain)
	if err != nil {
		hqgolog.Fatal().Msgf("%s", err)
	}

	options := &xcrawl3r.Options{
		Domain:            parsedURL.Domain,
		IncludeSubdomains: includeSubdomains,
		Seeds:             seeds,

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

	crawler, err := xcrawl3r.New(options)
	if err != nil {
		hqgolog.Fatal().Msgf("%s", err)
	}

	URLs := crawler.Crawl()

	if output != "" {
		directory := filepath.Dir(output)

		if _, err := os.Stat(directory); os.IsNotExist(err) {
			if err = os.MkdirAll(directory, os.ModePerm); err != nil {
				hqgolog.Fatal().Msg(err.Error())
			}
		}

		file, err := os.OpenFile(output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			hqgolog.Fatal().Msg(err.Error())
		}

		defer file.Close()

		writer := bufio.NewWriter(file)

		for outputURL := range URLs {
			if verbosity == string(levels.LevelSilent) {
				hqgolog.Print().Msg(outputURL.Value)
			} else {
				hqgolog.Print().Msgf("[%s] %s", au.BrightBlue(outputURL.Source), outputURL.Value)
			}

			fmt.Fprintln(writer, outputURL.Value)
		}

		if err = writer.Flush(); err != nil {
			hqgolog.Fatal().Msg(err.Error())
		}
	} else {
		for outputURL := range URLs {
			if verbosity == string(levels.LevelSilent) {
				hqgolog.Print().Msg(outputURL.Value)
			} else {
				hqgolog.Print().Msgf("[%s] %s", au.BrightBlue(outputURL.Source), outputURL.Value)
			}
		}
	}
}
