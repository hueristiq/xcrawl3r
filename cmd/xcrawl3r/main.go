package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"

	hqgologger "github.com/hueristiq/hq-go-logger"
	"github.com/hueristiq/hq-go-logger/formatter"
	"github.com/hueristiq/hq-go-logger/levels"
	"github.com/hueristiq/xcrawl3r/internal/configuration"
	"github.com/hueristiq/xcrawl3r/internal/input"
	"github.com/hueristiq/xcrawl3r/internal/output"
	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
	"github.com/logrusorgru/aurora/v4"
	"github.com/spf13/pflag"
)

var (
	inputURLs             []string
	inputURLsListFilePath string
	domains               []string
	includeSubdomains     bool
	depth                 int
	concurrency           int
	parallelism           int
	delay                 int
	headers               []string
	timeout               int
	proxies               []string
	debug                 bool
	outputInJSONL         bool
	monochrome            bool
	outputFilePath        string
	silent                bool
	verbose               bool

	au = aurora.New(aurora.WithColors(true))
)

func init() {
	defaultDepth := 1
	defaultConcurrency := 10
	defaultParallelism := 10
	defaultTimeout := 10

	pflag.StringSliceVarP(&inputURLs, "url", "u", []string{}, "")
	pflag.StringVarP(&inputURLsListFilePath, "list", "l", "", "")
	pflag.StringSliceVarP(&domains, "domain", "d", []string{}, "")
	pflag.BoolVar(&includeSubdomains, "include-subdomains", false, "")
	pflag.IntVar(&depth, "depth", defaultDepth, "")
	pflag.IntVarP(&concurrency, "concurrency", "c", defaultConcurrency, "")
	pflag.IntVarP(&parallelism, "parallelism", "p", defaultParallelism, "")
	pflag.IntVar(&delay, "delay", 0, "")
	pflag.StringSliceVarP(&headers, "header", "H", []string{}, "")
	pflag.IntVar(&timeout, "timeout", defaultTimeout, "")
	pflag.StringSliceVar(&proxies, "proxy", []string{}, "")
	pflag.BoolVar(&debug, "debug", false, "")
	pflag.BoolVar(&outputInJSONL, "jsonl", false, "")
	pflag.BoolVarP(&monochrome, "monochrome", "m", false, "")
	pflag.StringVarP(&outputFilePath, "output", "o", "", "")
	pflag.BoolVar(&silent, "silent", false, "")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "")

	pflag.Usage = func() {
		hqgologger.Info().Label("").Msg(configuration.BANNER(au))

		h := "USAGE:\n"
		h += fmt.Sprintf(" %s [OPTIONS]\n", configuration.NAME)

		h += "\nINPUT:\n"
		h += " -u, --url string[]                target URL\n"
		h += " -l, --list string                 target URLs list file path\n"

		h += "\nTIP: For multiple input URLs use comma(,) separated value with `-u`,\n"
		h += "     specify multiple `-u`, load from file with `-l` or load from stdin.\n"

		h += "\nSCOPE:\n"
		h += " -d, --domain string[]             domain to match URLs\n"

		h += "\nTIP: For multiple domains use comma(,) separated value with `-d`\n"
		h += "     or specify multiple `-d`.\n\n"

		h += "     --include-subdomains bool     with domain(s), match subdomains' URLs\n"

		h += "\nCONFIGURATION:\n"
		h += fmt.Sprintf("     --depth int                   maximum depth to crawl, `0` for infinite (default: %d)\n", defaultDepth)
		h += fmt.Sprintf(" -c, --concurrency int             number of concurrent inputs to process (default: %d)\n", defaultConcurrency)
		h += fmt.Sprintf(" -p, --parallelism int             number of concurrent fetchers to use (default: %d)\n", defaultParallelism)
		h += "     --delay int                   delay between each request in seconds\n"
		h += " -H, --header string[]             custom header to include in requests\n"

		h += "\nTIP: For multiple headers use comma(,) separated value with `--header`\n"
		h += "     or specify multiple `--header`.\n\n"

		h += fmt.Sprintf("     --timeout int                 time to wait for request in seconds (default: %d)\n", defaultTimeout)
		h += "     --proxy string[]              Proxy URL (e.g: http://127.0.0.1:8080)\n"

		h += "\nTIP: For multiple proxies use comma(,) separated value with `--proxy`\n"
		h += "     or specify multiple `--proxy`.\n"

		h += "\nOUTPUT:\n"
		h += "     --debug bool                  enable debug mode\n"
		h += "     --jsonl bool                  output URLs in JSONL format\n"
		h += " -m, --monochrome bool             stdout monochrome output\n"
		h += " -o, --output string               output URLs file path\n"
		h += " -s, --silent bool                 stdout URLs only output\n"
		h += " -v, --verbose bool                stdout verbose output\n"

		hqgologger.Info().Label("").Msg(h)
		hqgologger.Print().Msg("")
	}

	pflag.Parse()

	hqgologger.DefaultLogger.SetFormatter(formatter.NewConsoleFormatter(&formatter.ConsoleFormatterConfiguration{
		Colorize: !monochrome,
	}))

	if verbose {
		hqgologger.DefaultLogger.SetMaxLogLevel(levels.LevelDebug)
	}

	if silent {
		hqgologger.DefaultLogger.SetMaxLogLevel(levels.LevelSilent)
	}

	au = aurora.New(aurora.WithColors(!monochrome))
}

func main() {
	hqgologger.Info().Label("").Msg(configuration.BANNER(au))

	URLs := make(chan string, concurrency)

	go func() {
		defer close(URLs)

		if len(inputURLs) > 0 {
			for _, URL := range inputURLs {
				URLs <- URL
			}
		}

		if inputURLsListFilePath != "" {
			file, err := os.Open(inputURLsListFilePath)
			if err != nil {
				hqgologger.Error().Msg(err.Error())
			}

			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				URL := scanner.Text()

				if URL != "" {
					URLs <- URL
				}
			}

			if err := scanner.Err(); err != nil {
				hqgologger.Error().Msg(err.Error())
			}

			file.Close()
		}

		if input.HasStdin() {
			scanner := bufio.NewScanner(os.Stdin)

			for scanner.Scan() {
				URL := scanner.Text()

				if URL != "" {
					URLs <- URL
				}
			}

			if err := scanner.Err(); err != nil {
				hqgologger.Error().Msg(err.Error())
			}
		}
	}()

	outputs := []io.Writer{
		os.Stdout,
	}

	writer := output.NewWriter()

	if outputInJSONL {
		writer.SetFormatToJSONL()
	}

	if outputFilePath != "" {
		file, err := writer.CreateFile(outputFilePath)
		if err != nil {
			hqgologger.Error().Msg(err.Error())
		}

		outputs = append(outputs, file)
	}

	cfg := &xcrawl3r.Configuration{
		Domains:           domains,
		IncludeSubdomains: includeSubdomains,
		Depth:             depth,
		Parallelism:       parallelism,
		Delay:             delay,
		Headers:           headers,
		Timeout:           timeout,
		Proxies:           proxies,
		Debug:             debug,
	}

	crawler, err := xcrawl3r.New(cfg)
	if err != nil {
		hqgologger.Fatal().Msg(err.Error())
	}

	wg := &sync.WaitGroup{}

	for range concurrency {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for URL := range URLs {
				for result := range crawler.Crawl(URL) {
					for index := range outputs {
						output := outputs[index]

						switch result.Type {
						case xcrawl3r.ResultError:
							if verbose {
								hqgologger.Error().Msgf("%s: %s", result.Source, result.Error)
							}
						case xcrawl3r.ResultURL:
							if err := writer.Write(output, result); err != nil {
								hqgologger.Error().Msg(err.Error())
							}
						}
					}
				}
			}
		}()
	}

	wg.Wait()
}
