package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	"github.com/hueristiq/hqcrawl3r/internal/configuration"
	"github.com/hueristiq/hqcrawl3r/pkg/hqcrawl3r"
	"github.com/hueristiq/hqcrawl3r/pkg/utils/io"
	hqlog "github.com/hueristiq/hqgoutils/log"
	"github.com/hueristiq/hqgoutils/log/formatter"
	"github.com/hueristiq/hqgoutils/log/levels"
	hqurl "github.com/hueristiq/hqgoutils/url"
	"github.com/spf13/pflag"
)

var (
	targetURL, targetURLs          string
	includeSubdomains              bool
	depth                          int
	userAgent                      string
	headers                        []string
	timeout, delay, maxRandomDelay int
	proxy                          string
	parallelism, concurrency       int
	debug, monochrome              bool
	verbosity                      string
)

func init() {
	pflag.StringVarP(&targetURL, "url", "u", "", "")
	pflag.StringVarP(&targetURLs, "urls", "U", "", "")
	pflag.BoolVar(&includeSubdomains, "include-subs", false, "")
	pflag.IntVarP(&depth, "depth", "d", 2, "")
	pflag.StringVar(&userAgent, "user-agent", "web", "")
	pflag.StringSliceVarP(&headers, "headers", "H", []string{}, "")
	pflag.IntVar(&timeout, "timeout", 10, "")
	pflag.IntVar(&delay, "delay", 1, "")
	pflag.IntVar(&maxRandomDelay, "max-random-delay", 1, "")
	pflag.StringVar(&proxy, "proxy", "", "")
	pflag.IntVarP(&parallelism, "parallelism", "p", 10, "")
	pflag.IntVarP(&concurrency, "concurrency", "c", 10, "")
	pflag.BoolVar(&debug, "debug", false, "")
	pflag.BoolVarP(&monochrome, "monochrome", "m", false, "")
	pflag.StringVarP(&verbosity, "verbosity", "v", string(levels.LevelInfo), "")

	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		h := configuration.BANNER

		h += "\nUSAGE:\n"
		h += "  hqcrawl3r [OPTIONS]\n"

		h += "\nOPTIONS:\n"
		h += "  -u, --url                  target URL\n"
		h += "  -U, --urls                 target URLs\n"
		h += "      --include-subs         extend scope to include subdomains (default: false)\n"
		h += "  -d, --depth                maximum recursion depth on visited URLs. (default: 1)\n"
		h += "                                 TIP: set it to `0` for infinite recursion\n"
		h += "      --user-agent           User Agent to use (default: web)\n"
		h += "                                 TIP: use `web` for a random web user-agent,\n"
		h += "                                 `mobile` for a random mobile user-agent,\n"
		h += "                                  or you can set your specific user-agent.\n"
		h += "  -H, --headers              custom header to include in requests\n"
		h += "                                 e.g. -H 'Referer: http://example.com/'\n"
		h += "                                 TIP: use multiple flag to set multiple headers\n"
		h += "      --timeout              time to wait for request in seconds (default: 10)\n"
		h += "      --delay                delay between request to matching domains (default: 1s)\n"
		h += "      --max-random-delay     extra randomized delay added to `--dalay` (default: 1s)\n"
		h += "      --proxy                Proxy URL (e.g: http://127.0.0.1:8080)\n"
		h += "  -p, --parallelism          number of concurrent URLs to process (default: 10)\n"
		h += "  -c, --concurrency          number of concurrent requests for matching domains (default: 10)\n"
		h += "      --debug                enable debug mode (default: false)\n"
		h += "  -m, --monochrome           coloring: no colored output mode\n"
		h += "  -v, --verbosity            debug, info, warning, error, fatal or silent (default: debug)\n"

		fmt.Fprint(os.Stderr, h)
	}

	pflag.Parse()

	hqlog.DefaultLogger.SetMaxLevel(levels.LevelStr(verbosity))
	hqlog.DefaultLogger.SetFormatter(formatter.NewCLI(&formatter.CLIOptions{
		Colorize: !monochrome,
	}))
}

func main() {
	var (
		err error
	)

	fmt.Fprintln(os.Stderr, configuration.BANNER)

	URLs := []string{}

	if targetURL != "" {
		URLs = append(URLs, targetURL)
	}

	if targetURLs != "" {
		var (
			file *os.File
		)

		switch {
		case targetURLs == "-" && io.HasStdIn():
			file = os.Stdin
		case targetURLs != "-":
			file, err = os.Open(targetURLs)
			if err != nil {
				hqlog.Fatal().Msgf("%s", err)
			}
		default:
			hqlog.Fatal().Msg("hqurlscann3r takes input from stdin or file using '-d' flag")
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			URL := scanner.Text()

			if URL != "" {
				URLs = append(URLs, URL)
			}
		}

		if scanner.Err() != nil {
			hqlog.Fatal().Msgf("%s", err)
		}
	}

	URLsCH := make(chan string, parallelism)
	URLsWG := new(sync.WaitGroup)

	for i := 0; i < parallelism; i++ {
		URLsWG.Add(1)

		go func() {
			defer URLsWG.Done()

			for URL := range URLsCH {
				parsedURL, err := hqurl.Parse(URL)
				if err != nil {
					hqlog.Error().Msgf("%s", err)

					continue
				}

				options := &hqcrawl3r.Options{
					TargetURL:         parsedURL,
					Concurrency:       concurrency,
					Debug:             debug,
					Depth:             depth,
					Headers:           headers,
					IncludeSubdomains: includeSubdomains,
					Delay:             delay,
					MaxRandomDelay:    maxRandomDelay,
					Proxy:             proxy,
					Timeout:           timeout,
					UserAgent:         userAgent,
				}

				crawler, err := hqcrawl3r.New(options)
				if err != nil {
					hqlog.Error().Msgf("%s", err)

					continue
				}

				wg := new(sync.WaitGroup)

				wg.Add(1)
				go func() {
					defer wg.Done()

					_, err = crawler.Run()
					if err != nil {
						hqlog.Error().Msgf("%s", err)
					}
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()

					crawler.ParseSitemap()
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()

					crawler.ParseRobots()
				}()

				wg.Wait()
			}
		}()
	}

	for index := range URLs {
		URLsCH <- URLs[index]
	}

	close(URLsCH)

	URLsWG.Wait()
}
