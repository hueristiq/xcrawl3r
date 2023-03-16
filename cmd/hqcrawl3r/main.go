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
	includeSubdomains     bool
	proxy                 string
	maxRandomDelay        int
	threads               int
	timeout               int
	userAgent             string
	targetURL, targetURLs string
	monochrome            bool
	verbosity             string
)

func init() {
	pflag.IntVarP(&concurrency, "concurrency", "c", 5, "")
	pflag.StringVar(&cookies, "cookie", "", "")
	pflag.BoolVar(&debug, "debug", false, "")
	pflag.IntVarP(&depth, "depth", "d", 2, "")
	pflag.StringVarP(&headers, "headers", "H", "", "")
	pflag.BoolVar(&includeSubdomains, "include-subs", false, "")
	pflag.StringVarP(&proxy, "proxy", "p", "", "")
	pflag.IntVarP(&maxRandomDelay, "random-delay", "R", 60, "")
	pflag.IntVar(&threads, "threads", 20, "")
	pflag.IntVar(&timeout, "timeout", 10, "")
	pflag.StringVarP(&targetURL, "url", "u", "", "")
	pflag.StringVarP(&targetURLs, "urls", "U", "", "")
	pflag.StringVar(&userAgent, "user-agent", "web", "")
	pflag.BoolVarP(&monochrome, "monochrome", "m", false, "")
	pflag.StringVarP(&verbosity, "verbosity", "v", string(levels.LevelInfo), "")

	pflag.CommandLine.SortFlags = false
	pflag.Usage = func() {
		h := configuration.BANNER

		h += "\nUSAGE:\n"
		h += "  hqcrawl3r [OPTIONS]\n"

		h += "\nOPTIONS:\n"
		h += "  -c, --concurrency          Maximum concurrent requests for matching domains (default: 5)\n"
		h += "      --cookie               Cookie to use (testA=a; testB=b)\n"
		h += "      --debug                Enable debug mode (default: false)\n"
		h += "  -d, --depth                Maximum recursion depth on visited URLs. (default: 1)\n"
		h += "                                 Note: Requires '-r, --render' flag\n"
		h += "                                 Note: Usage to show browser: '--headless=false' (default true)\n"
		h += "  -H, --headers              Custom headers separated by two semi-colons.\n"
		h += "                                 E.g. -h 'Cookie: foo=bar;;Referer: http://example.com/'\n"
		h += "      --include-subs         Extend scope to include subdomains (default: false)\n"
		h += "  -p, --proxy                Proxy URL (e.g: http://127.0.0.1:8080)\n"
		h += "  -R, --random-delay         Maximum random delay between requests (default: 2s)\n"
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

	URLsCH := make(chan string, threads)
	URLsWG := new(sync.WaitGroup)

	for i := 0; i < threads; i++ {
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
					Cookie:            cookies,
					Debug:             debug,
					Depth:             depth,
					Headers:           headers,
					IncludeSubdomains: includeSubdomains,
					MaxRandomDelay:    maxRandomDelay,
					Proxy:             proxy,
					Threads:           threads,
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

					crawler.Crawl()
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
