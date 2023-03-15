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
	concurrency           int
	cookies               string
	debug                 bool
	depth                 int
	headers               string
	headless              bool
	includeSubdomains     bool
	proxy                 string
	maxRandomDelay        int
	render                bool
	threads               int
	timeout               int
	userAgent             string
	targetURL, targetURLs string
	monochrome            bool
	verbosity             string
)

func displayBanner() {
	fmt.Fprintln(os.Stderr, configuration.BANNER)
}

func init() {
	pflag.IntVarP(&concurrency, "concurrency", "c", 5, "")
	pflag.StringVar(&cookies, "cookie", "", "")
	pflag.BoolVar(&debug, "debug", false, "")
	pflag.IntVarP(&depth, "depth", "d", 1, "")
	pflag.StringVarP(&headers, "headers", "H", "", "")
	pflag.BoolVar(&headless, "headless", true, "")
	pflag.BoolVar(&includeSubdomains, "include-subs", false, "")
	pflag.StringVarP(&proxy, "proxy", "p", "", "")
	pflag.IntVarP(&maxRandomDelay, "random-delay", "R", 60, "")
	pflag.BoolVarP(&render, "render", "r", false, "")
	pflag.IntVar(&threads, "threads", 20, "")
	pflag.IntVar(&timeout, "timeout", 10, "")
	pflag.StringVarP(&targetURL, "url", "u", "", "")
	pflag.StringVarP(&targetURLs, "urls", "U", "", "")
	pflag.StringVar(&userAgent, "user-agent", "web", "")
	pflag.BoolVarP(&monochrome, "monochrome", "m", false, "")
	pflag.StringVarP(&verbosity, "verbosity", "v", string(levels.LevelInfo), "")

	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		displayBanner()

		h := "USAGE:\n"
		h += "  hqcrawl3r [OPTIONS]\n"

		h += "\nOPTIONS:\n"
		h += "  -c, --concurrency          Maximum concurrent requests for matching domains (default: 5)\n"
		h += "      --cookie               Cookie to use (testA=a; testB=b)\n"
		h += "      --debug                Enable debug mode (default: false)\n"
		h += "  -d, --depth                Maximum recursion depth on visited URLs. (default: 1)\n"
		h += "      --headless             If true the browser will be displayed while crawling\n"
		h += "                                 Note: Requires '-r, --render' flag\n"
		h += "                                 Note: Usage to show browser: '--headless=false' (default true)\n"
		h += "  -H, --headers              Custom headers separated by two semi-colons.\n"
		h += "                                 E.g. -h 'Cookie: foo=bar;;Referer: http://example.com/'\n"
		h += "      --include-subs         Extend scope to include subdomains (default: false)\n"
		h += "  -p, --proxy                Proxy URL (e.g: http://127.0.0.1:8080)\n"
		h += "  -R, --random-delay         Maximum random delay between requests (default: 2s)\n"
		h += "  -r, --render               Render javascript.\n"
		h += "  -t, --threads              Number of threads (Run URLs in parallel) (default: 20)\n"
		h += "      --timeout              Request timeout (second) (default: 10)\n"
		h += "  -u, --url                  URL to crawl\n"
		h += "  -U, --urls                 URLs to crawl\n"
		h += "      --user-agent           User Agent to use (default: web)\n"
		h += "                                 `web` for a random web user-agent\n"
		h += "                                 `mobile` for a random mobile user-agent\n"
		h += "                                 or you can set your special user-agent\n"
		h += "  -m, --monochrome                coloring: no colored output mode\n"
		h += "  -v, --verbosity                 debug, info, warning, error, fatal or silent (default: debug)\n"

		fmt.Fprint(os.Stderr, h)
	}

	pflag.Parse()

	hqlog.DefaultLogger.SetMaxLevel(levels.LevelStr(verbosity))
	hqlog.DefaultLogger.SetFormatter(formatter.NewCLI(&formatter.CLIOptions{
		Colorize: !monochrome,
	}))
}

func main() {
	displayBanner()

	var (
		f       *os.File
		err     error
		scanner *bufio.Scanner
	)

	URLs := []string{}

	// input: URL
	if targetURL != "" {
		URLs = append(URLs, targetURL)
	}

	// input: Stdin
	if io.HasStdIn() {
		f = os.Stdin

		scanner = bufio.NewScanner(f)

		for scanner.Scan() {
			URL := scanner.Text()

			if URL != "" {
				URLs = append(URLs, URL)
			}
		}

		if err = scanner.Err(); err != nil {
			hqlog.Fatal().Msgf("%s", err)
		}
	}

	// input: URLs File
	if targetURLs != "" {
		f, err = os.Open(targetURLs)
		if err != nil {
			hqlog.Fatal().Msgf("%s", err)
		}

		scanner = bufio.NewScanner(f)

		for scanner.Scan() {
			URL := scanner.Text()

			if URL != "" {
				URLs = append(URLs, URL)
			}
		}

		if err = scanner.Err(); err != nil {
			hqlog.Fatal().Msgf("%s", err)
		}
	}

	wg := new(sync.WaitGroup)
	inputURLsChan := make(chan string, threads)

	for i := 0; i < threads; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for URL := range inputURLsChan {
				parsedURL, err := hqurl.Parse(URL)
				if err != nil {
					hqlog.Error().Msgf("%s", err)
					continue
				}

				URLswg := new(sync.WaitGroup)

				options := &hqcrawl3r.Options{
					TargetURL:         parsedURL,
					Concurrency:       concurrency,
					Cookie:            cookies,
					Debug:             debug,
					Depth:             depth,
					Headers:           headers,
					Headless:          headless,
					IncludeSubdomains: includeSubdomains,
					MaxRandomDelay:    maxRandomDelay,
					Proxy:             proxy,
					Render:            render,
					Threads:           threads,
					Timeout:           timeout,
					UserAgent:         userAgent,
				}

				crawler, err := hqcrawl3r.New(options)
				if err != nil {
					hqlog.Error().Msgf("%s", err)
					continue
				}

				// crawl
				URLswg.Add(1)
				go func() {
					defer URLswg.Done()

					crawler.Crawl()
				}()

				// parse sitemaps
				URLswg.Add(1)
				go func() {
					defer URLswg.Done()

					crawler.ParseSitemap()
				}()

				// parse robots.txt
				URLswg.Add(1)
				go func() {
					defer URLswg.Done()

					crawler.ParseRobots()
				}()

				URLswg.Wait()
			}
		}()
	}

	for _, URL := range URLs {
		inputURLsChan <- URL
	}

	close(inputURLsChan)

	wg.Wait()
}
