package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/enenumxela/urlx/pkg/urlx"
	"github.com/logrusorgru/aurora/v3"
	"github.com/signedsecurity/sigrawl3r/internal/configuration"
	"github.com/signedsecurity/sigrawl3r/internal/crawler"
	"github.com/signedsecurity/sigrawl3r/internal/utils/io"
)

var (
	au       aurora.Aurora
	conf     configuration.Configuration
	URL      string
	URLsFile string
	silent   bool
	noColor  bool
)

func displayBanner() {
	fmt.Fprintln(os.Stderr, configuration.BANNER)
}

func init() {
	flag.IntVar(&conf.Concurrency, "concurrency", configuration.DefaultConcurrency, "")
	flag.IntVar(&conf.Concurrency, "c", configuration.DefaultConcurrency, "")
	flag.StringVar(&conf.Cookie, "cookie", "", "")
	flag.BoolVar(&conf.Debug, "debug", false, "")
	flag.IntVar(&conf.Depth, "depth", configuration.DefaultDepth, "")
	flag.IntVar(&conf.Depth, "d", configuration.DefaultDepth, "")
	flag.StringVar(&conf.Headers, "headers", "", "")
	flag.StringVar(&conf.Headers, "H", "", "")
	flag.BoolVar(&conf.Headless, "headless", true, "")
	flag.BoolVar(&conf.IncludeSubdomains, "include-subs", false, "")
	flag.BoolVar(&noColor, "no-color", false, "")
	flag.StringVar(&conf.Proxy, "proxy", "", "")
	flag.StringVar(&conf.Proxy, "p", "", "")
	flag.IntVar(&conf.MaxRandomDelay, "random-delay", configuration.DefaultMaxRandomDelay, "")
	flag.IntVar(&conf.MaxRandomDelay, "R", configuration.DefaultMaxRandomDelay, "")
	flag.BoolVar(&conf.Render, "render", false, "")
	flag.BoolVar(&conf.Render, "r", false, "")
	flag.BoolVar(&silent, "silent", false, "")
	flag.BoolVar(&silent, "s", false, "")
	flag.IntVar(&conf.Threads, "threads", configuration.DefaultThreads, "")
	flag.IntVar(&conf.Timeout, "timeout", configuration.DefaultTimeout, "")
	flag.StringVar(&URL, "url", "", "")
	flag.StringVar(&URL, "u", "", "")
	flag.StringVar(&URLsFile, "urls", "", "")
	flag.StringVar(&URLsFile, "U", "", "")
	flag.StringVar(&conf.UserAgent, "user-agent", "web", "")

	flag.Usage = func() {
		displayBanner()

		h := "USAGE:\n"
		h += "  sigrawl3r [OPTIONS]\n"

		h += "\nOPTIONS:\n"
		h += fmt.Sprintf("  -c, --concurrency          Maximum concurrent requests for matching domains (default: %d)\n", configuration.DefaultConcurrency)
		h += "      --cookie               Cookie to use (testA=a; testB=b)\n"
		h += "      --debug                Enable debug mode (default: false)\n"
		h += fmt.Sprintf("  -d, --depth                Maximum recursion depth on visited URLs. (default: %d)\n", configuration.DefaultDepth)
		h += "      --headless             If true the browser will be displayed while crawling\n"
		h += "                                 Note: Requires '-r, --render' flag\n"
		h += "                                 Note: Usage to show browser: '--headless=false' (default true)\n"
		h += "  -H, --headers              Custom headers separated by two semi-colons. E.g. -h 'Cookie: foo=bar;;Referer: http://example.com/'\n"
		h += "      --include-subs         Extend scope to include subdomains (default: false)\n"
		h += "      --no-color             Enable no color mode (default: false)\n"
		h += "  -p, --proxy                Proxy URL (e.g: http://127.0.0.1:8080)\n"
		h += "  -R, --random-delay         Maximum random delay between requests (default: 2s)\n"
		h += "  -r, --render               Render javascript.\n"
		h += "  -s, --silent               Enable silent mode: output urls only (default: false)\n"
		h += fmt.Sprintf("  -t, --threads              Number of threads (Run URLs in parallel) (default: %d)\n", configuration.DefaultThreads)
		h += fmt.Sprintf("      --timeout              Request timeout (second) (default: %d)\n", configuration.DefaultTimeout)
		h += "  -u, --url                  URL to crawl\n"
		h += "  -U, --urls                 URLs to crawl\n"
		h += "      --user-agent           User Agent to use (default: web)\n"
		h += "                                 `web` for a random web user-agent\n"
		h += "                                 `mobile` for a random mobile user-agent\n"
		h += "                                 or you can set your special user-agent\n"

		fmt.Fprint(os.Stderr, h)
	}

	flag.Parse()

	au = aurora.NewAurora(!noColor)
}

func main() {
	if !silent {
		displayBanner()
	}

	// validate configuration
	if err := conf.Validate(); err != nil {
		log.Fatalln(err)
	}

	var (
		f       *os.File
		err     error
		scanner *bufio.Scanner
	)

	URLs := []string{}

	// input: URL
	if URL != "" {
		URLs = append(URLs, URL)
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
			log.Fatalln(err)
		}
	}

	// input: URLs File
	if URLsFile != "" {
		f, err = os.Open(URLsFile)
		if err != nil {
			log.Fatalln(err)
		}

		scanner = bufio.NewScanner(f)

		for scanner.Scan() {
			URL := scanner.Text()

			if URL != "" {
				URLs = append(URLs, URL)
			}
		}

		if err = scanner.Err(); err != nil {
			log.Fatalln(err)
		}
	}

	// process URLs
	inputURLsChan := make(chan string, conf.Threads)

	wg := new(sync.WaitGroup)

	for i := 0; i < conf.Threads; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for URL := range inputURLsChan {
				parsedURL, err := urlx.Parse(URL)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}

				URLwg := new(sync.WaitGroup)

				c, err := crawler.New(parsedURL, &conf)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}

				// parse robots.txt
				URLwg.Add(1)
				go func() {
					defer URLwg.Done()

					c.ParseRobots()
				}()

				// parse sitemaps
				URLwg.Add(1)
				go func() {
					defer URLwg.Done()

					c.ParseSitemap()
				}()

				// crawl
				URLwg.Add(1)
				go func() {
					defer URLwg.Done()

					c.Run()
				}()

				URLwg.Wait()
			}
		}()
	}

	for _, URL := range URLs {
		inputURLsChan <- URL
	}

	close(inputURLsChan)
	wg.Wait()
}
