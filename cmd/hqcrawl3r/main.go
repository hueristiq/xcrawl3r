package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/hueristiq/hqcrawl3r/internal/configuration"
	"github.com/hueristiq/hqcrawl3r/internal/crawler"
	"github.com/hueristiq/hqcrawl3r/internal/utils/io"
	"github.com/hueristiq/url"
	"github.com/logrusorgru/aurora/v3"
)

var (
	au       aurora.Aurora
	c        configuration.Configuration
	noColor  bool
	silent   bool
	URL      string
	URLsFile string
)

func displayBanner() {
	fmt.Fprintln(os.Stderr, configuration.BANNER)
}

func init() {
	flag.IntVar(&c.Concurrency, "concurrency", configuration.DefaultConcurrency, "")
	flag.IntVar(&c.Concurrency, "c", configuration.DefaultConcurrency, "")
	flag.StringVar(&c.Cookie, "cookie", "", "")
	flag.BoolVar(&c.Debug, "debug", false, "")
	flag.IntVar(&c.Depth, "depth", configuration.DefaultDepth, "")
	flag.IntVar(&c.Depth, "d", configuration.DefaultDepth, "")
	flag.StringVar(&c.Headers, "headers", "", "")
	flag.StringVar(&c.Headers, "H", "", "")
	flag.BoolVar(&c.Headless, "headless", true, "")
	flag.BoolVar(&c.IncludeSubdomains, "include-subs", false, "")
	flag.BoolVar(&noColor, "no-color", false, "")
	flag.StringVar(&c.Proxy, "proxy", "", "")
	flag.StringVar(&c.Proxy, "p", "", "")
	flag.IntVar(&c.MaxRandomDelay, "random-delay", configuration.DefaultMaxRandomDelay, "")
	flag.IntVar(&c.MaxRandomDelay, "R", configuration.DefaultMaxRandomDelay, "")
	flag.BoolVar(&c.Render, "render", false, "")
	flag.BoolVar(&c.Render, "r", false, "")
	flag.BoolVar(&silent, "silent", false, "")
	flag.BoolVar(&silent, "s", false, "")
	flag.IntVar(&c.Threads, "threads", configuration.DefaultThreads, "")
	flag.IntVar(&c.Timeout, "timeout", configuration.DefaultTimeout, "")
	flag.StringVar(&URL, "url", "", "")
	flag.StringVar(&URL, "u", "", "")
	flag.StringVar(&URLsFile, "urls", "", "")
	flag.StringVar(&URLsFile, "U", "", "")
	flag.StringVar(&c.UserAgent, "user-agent", "web", "")

	flag.Usage = func() {
		displayBanner()

		h := "USAGE:\n"
		h += "  hqcrawl3r [OPTIONS]\n"

		h += "\nOPTIONS:\n"
		h += fmt.Sprintf("  -c, --concurrency          Maximum concurrent requests for matching domains (default: %d)\n", configuration.DefaultConcurrency)
		h += "      --cookie               Cookie to use (testA=a; testB=b)\n"
		h += "      --debug                Enable debug mode (default: false)\n"
		h += fmt.Sprintf("  -d, --depth                Maximum recursion depth on visited URLs. (default: %d)\n", configuration.DefaultDepth)
		h += "      --headless             If true the browser will be displayed while crawling\n"
		h += "                                 Note: Requires '-r, --render' flag\n"
		h += "                                 Note: Usage to show browser: '--headless=false' (default true)\n"
		h += "  -H, --headers              Custom headers separated by two semi-colons.\n"
		h += "                                 E.g. -h 'Cookie: foo=bar;;Referer: http://example.com/'\n"
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
		h += "\n"

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
	if err := c.Validate(); err != nil {
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

	wg := new(sync.WaitGroup)
	inputURLsChan := make(chan string, c.Threads)

	for i := 0; i < c.Threads; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for URL := range inputURLsChan {
				parsedURL, err := url.Parse(url.Options{URL: URL})
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}

				URLswg := new(sync.WaitGroup)

				c, err := crawler.New(parsedURL, &c)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}

				// crawl
				URLswg.Add(1)
				go func() {
					defer URLswg.Done()

					c.Crawl()
				}()

				// parse sitemaps
				URLswg.Add(1)
				go func() {
					defer URLswg.Done()

					c.ParseSitemap()
				}()

				// parse robots.txt
				URLswg.Add(1)
				go func() {
					defer URLswg.Done()

					c.ParseRobots()
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
