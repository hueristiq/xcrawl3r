package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	hqgohttpheader "github.com/hueristiq/hq-lib-http-go/header"
	hqgologger "github.com/hueristiq/hq-lib-logger-go"
	hqgologgerformatter "github.com/hueristiq/hq-lib-logger-go/formatter"
	hqgologgerlevels "github.com/hueristiq/hq-lib-logger-go/levels"
	"github.com/logrusorgru/aurora/v4"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/hueristiq/xcrawl3r/internal/configuration"
	"github.com/hueristiq/xcrawl3r/internal/input"
	"github.com/hueristiq/xcrawl3r/internal/output"
	"github.com/hueristiq/xcrawl3r/pkg/xcrawl3r"
)

var (
	configurationFilePath string
	urls                  []string
	urlsListFilePath      string
	domains               []string
	includeSubdomains     bool
	delay                 int
	randomDelay           int
	headers               []string
	cookies               []string
	burpFilePath          string
	timeout               int
	retries               int
	requestsPerMinute     int
	randomAgent           bool
	proxies               []string
	depth                 int
	concurrency           int
	parallelism           int
	maxPages              int
	crawlDuration         string
	ignoreQuery           bool
	shapeDedup            bool
	headless              bool
	headlessTimeout       int
	debug                 bool
	outputInJSONL         bool
	outputFilePath        string
	monochrome            bool
	silent                bool
	verbose               bool

	au = aurora.New(aurora.WithColors(true))
)

func init() {
	pflag.StringVarP(&configurationFilePath, "configuration", "c", configuration.DefaultConfigurationFilePath, "")
	pflag.StringSliceVarP(&urls, "url", "u", []string{}, "")
	pflag.StringVarP(&urlsListFilePath, "list", "l", "", "")
	pflag.StringSliceVarP(&domains, "domain", "d", []string{}, "")
	pflag.BoolVar(&includeSubdomains, "include-subdomains", false, "")
	pflag.IntVar(&delay, "delay", configuration.DefaultConfiguration.Request.Delay, "")
	pflag.IntVar(&randomDelay, "random-delay", configuration.DefaultConfiguration.Request.RandomDelay, "")
	pflag.StringSliceVarP(&headers, "header", "H", []string{}, "")
	pflag.StringSliceVar(&cookies, "cookie", []string{}, "")
	pflag.StringVar(&burpFilePath, "burp", "", "")
	pflag.IntVar(&timeout, "timeout", configuration.DefaultConfiguration.Request.Timeout, "")
	pflag.IntVar(&retries, "retries", configuration.DefaultConfiguration.Request.Retries, "")
	pflag.IntVar(&requestsPerMinute, "requests-per-minute", configuration.DefaultConfiguration.Request.RequestsPerMinute, "")
	pflag.BoolVar(&randomAgent, "random-agent", false, "")
	pflag.StringSliceVarP(&proxies, "proxy", "p", []string{}, "")
	pflag.IntVar(&depth, "depth", configuration.DefaultConfiguration.Optimization.Depth, "")
	pflag.IntVarP(&concurrency, "concurrency", "C", configuration.DefaultConfiguration.Optimization.Concurrency, "")
	pflag.IntVarP(&parallelism, "parallelism", "P", configuration.DefaultConfiguration.Optimization.Parallelism, "")
	pflag.IntVar(&maxPages, "max-pages", configuration.DefaultConfiguration.Optimization.MaxPages, "")
	pflag.StringVar(&crawlDuration, "crawl-duration", configuration.DefaultConfiguration.Optimization.CrawlDuration, "")
	pflag.BoolVar(&ignoreQuery, "ignore-query", false, "")
	pflag.BoolVar(&shapeDedup, "shape-dedup", false, "")
	pflag.BoolVar(&headless, "headless", false, "")
	pflag.IntVar(&headlessTimeout, "headless-timeout", 10, "")
	pflag.BoolVar(&debug, "debug", false, "")
	pflag.BoolVar(&outputInJSONL, "jsonl", false, "")
	pflag.StringVarP(&outputFilePath, "output", "o", "", "")
	pflag.BoolVarP(&monochrome, "monochrome", "m", false, "")
	pflag.BoolVar(&silent, "silent", false, "")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "")

	pflag.Usage = func() {
		hqgologger.Info(configuration.Banner(au), hqgologger.WithoutTimestamp(), hqgologger.WithoutLabel())

		h := "USAGE:\n"
		h += fmt.Sprintf(" %s [OPTIONS]\n", configuration.Name)

		h += "\nCONFIGURATION:\n"

		defaultConfigurationFilePath := strings.ReplaceAll(configuration.DefaultConfigurationFilePath, configuration.UserDotConfigDirectoryPath, "$HOME/.config")

		h += fmt.Sprintf(" -c, --configuration string       (default: %v)\n", au.Underline(defaultConfigurationFilePath).Bold())

		h += "\nINPUT:\n"
		h += " -u, --url string[]               target URL\n"
		h += " -l, --list string                target URLs file path\n"

		h += "\n For multiple URLs, use comma(,) separated value with `--url`,\n"
		h += " specify multiple `--url`, load from file with `--list` or load from stdin.\n"

		h += "\nSCOPE:\n"
		h += " -d, --domain string[]            match domain(s)  URLs\n"

		h += "\n For multiple domains, use comma(,) separated value with `--domain`\n"
		h += " or specify multiple `--domain`.\n\n"

		h += "     --include-subdomains bool    with domain(s), match subdomains' URLs\n"

		h += "\n Without `--domain`, only URLs on the target's host are matched.\n"

		h += "\nREQUEST:\n"
		h += "     --delay int                  delay between each request in seconds\n"
		h += "     --random-delay int           random delay between each request in seconds\n"
		h += " -H, --header string[]            header to include in 'header:value' format\n"

		h += "\n For multiple headers, use comma(,) separated value with `--header`\n"
		h += " or specify multiple `--header`.\n\n"

		h += fmt.Sprintf("     --timeout int                time to wait for request in seconds (default: %d)\n", configuration.DefaultConfiguration.Request.Timeout)
		h += fmt.Sprintf("     --retries int                number of retries for failed requests (default: %d)\n", configuration.DefaultConfiguration.Request.Retries)
		h += "     --random-agent bool          use a random User-Agent for each request\n"
		h += "     --requests-per-minute int    maximum requests per minute, per host (default: 0, no limit)\n"
		h += "     --cookie string[]            cookie to include in 'name=value' format\n"
		h += "     --burp string                burp request file path, headers & cookies are imported\n"

		h += "\nPROXY:\n"
		h += " -p, --proxy string[]             Proxy (e.g: http://127.0.0.1:8080)\n"

		h += "\n For multiple proxies use comma(,) separated value with `--proxy`\n"
		h += " or specify multiple `--proxy`.\n"

		h += "\nOPTIMIZATION:\n"
		h += fmt.Sprintf("     --depth int                  maximum depth to crawl, `0` for infinite (default: %d)\n", configuration.DefaultConfiguration.Optimization.Depth)
		h += fmt.Sprintf(" -C, --concurrency int            number of concurrent inputs to process (default: %d)\n", configuration.DefaultConfiguration.Optimization.Concurrency)
		h += fmt.Sprintf(" -P, --parallelism int            number of concurrent fetchers to use (default: %d)\n", configuration.DefaultConfiguration.Optimization.Parallelism)
		h += "     --max-pages int              maximum pages to crawl per target, `0` for no limit (default: 0)\n"
		h += "     --crawl-duration duration    maximum duration per target crawl (e.g: 10s, 5m), `0` for no limit (default: 0)\n"
		h += "     --ignore-query bool          deduplicate URLs ignoring the query string\n"
		h += "     --shape-dedup bool           deduplicate URLs by shape (numbers, UUIDs, dates normalized)\n"

		h += "\nDEBUG:\n"
		h += "     --debug bool                 enable debug mode\n"

		h += "\nHEADLESS:\n"
		h += "     --headless bool              render pages in a headless browser, requires Chrome/Chromium\n"
		h += "     --headless-timeout int       time to wait for page rendering in seconds (default: 10)\n"

		h += "\nOUTPUT:\n"
		h += "     --jsonl bool                 output in JSONL(ines)\n"
		h += " -o, --output string              output write file path\n"
		h += " -m, --monochrome bool            disable colored console output\n"
		h += " -s, --silent bool                disable logging output, only results\n"
		h += " -v, --verbose bool               enable detailed debug logging output\n"

		hqgologger.Info(h, hqgologger.WithoutTimestamp(), hqgologger.WithoutLabel())
		hqgologger.Print("", hqgologger.WithoutTimestamp(), hqgologger.WithoutLabel())
	}

	pflag.Parse()

	if err := configuration.CreateOrUpdate(configurationFilePath); err != nil {
		hqgologger.Fatal("failed creating or updating Configuration!", hqgologger.WithError(err))
	}

	viper.SetConfigFile(configurationFilePath)

	viper.AutomaticEnv()

	viper.SetEnvPrefix(strings.ToUpper(configuration.Name))
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		hqgologger.Fatal("failed reading in Configuration!", hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("request.delay", pflag.Lookup("delay")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "delay"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("request.random_delay", pflag.Lookup("random-delay")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "random-delay"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("request.timeout", pflag.Lookup("timeout")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "timeout"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("request.retries", pflag.Lookup("retries")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "retries"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("request.requests_per_minute", pflag.Lookup("requests-per-minute")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "requests-per-minute"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("optimization.depth", pflag.Lookup("depth")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "depth"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("optimization.concurrency", pflag.Lookup("concurrency")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "concurrency"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("optimization.parallelism", pflag.Lookup("parallelism")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "parallelism"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("optimization.max_pages", pflag.Lookup("max-pages")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "max-pages"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("optimization.crawl_duration", pflag.Lookup("crawl-duration")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "crawl-duration"), hqgologger.WithError(err))
	}

	hqgologger.DefaultLogger.SetFormatter(hqgologgerformatter.NewConsoleFormatter(&hqgologgerformatter.ConsoleFormatterConfiguration{
		Colorize: !monochrome,
	}))

	if silent {
		if err := hqgologger.DefaultLogger.SetLevel(hqgologgerlevels.LevelSilent); err != nil {
			hqgologger.Fatal("failed setting logger level!", hqgologger.WithError(err))
		}
	}

	if verbose {
		if err := hqgologger.DefaultLogger.SetLevel(hqgologgerlevels.LevelDebug); err != nil {
			hqgologger.Fatal("failed setting logger level!", hqgologger.WithError(err))
		}
	}

	au = aurora.New(aurora.WithColors(!monochrome))
}

func main() {
	hqgologger.Info(configuration.Banner(au), hqgologger.WithLabel(""))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	c := viper.GetInt("optimization.concurrency")
	if c < 1 {
		// Zero or negative concurrency would start no workers (or panic on
		// make); clamp to a single worker.
		c = 1
	}

	urlsChan := make(chan string, c)

	go func() {
		defer close(urlsChan)

		send := func(URL string) (sent bool) {
			select {
			case urlsChan <- URL:
				return true
			case <-ctx.Done():
				return false
			}
		}

		for _, URL := range urls {
			if !send(URL) {
				return
			}
		}

		if urlsListFilePath != "" {
			//nolint:gosec // path is the user-provided input file path.
			file, err := os.Open(urlsListFilePath)
			if err != nil {
				hqgologger.Fatal("failed opening input file", hqgologger.WithError(err))
			}

			defer func() {
				_ = file.Close()
			}()

			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				if URL := scanner.Text(); URL != "" && !send(URL) {
					return
				}
			}

			if err := scanner.Err(); err != nil {
				hqgologger.Fatal("failed reading input file!", hqgologger.WithError(err))
			}
		}

		if input.HasStdin() {
			scanner := bufio.NewScanner(os.Stdin)

			for scanner.Scan() {
				if URL := scanner.Text(); URL != "" && !send(URL) {
					return
				}
			}

			if err := scanner.Err(); err != nil {
				hqgologger.Fatal("failed reading stdin!", hqgologger.WithError(err))
			}
		}
	}()

	writer := output.NewWriter()

	if outputInJSONL {
		writer.SetFormatToJSONL()
	}

	outputs := []io.Writer{
		os.Stdout,
	}

	if outputFilePath != "" {
		file, err := writer.CreateFile(outputFilePath)
		if err != nil {
			hqgologger.Fatal("failed creating output file!", hqgologger.WithString("file", outputFilePath), hqgologger.WithError(err))
		}

		defer func() {
			_ = file.Close()
		}()

		outputs = append(outputs, file)
	}

	h := viper.GetStringSlice("request.headers")

	if !randomAgent {
		h = append(h, []string{
			fmt.Sprintf("%s: %s %s (https://github.com/hueristiq/%s.git)", hqgohttpheader.UserAgent.String(), configuration.Name, configuration.Version, configuration.Name),
		}...)
	}

	if burpFilePath != "" {
		var burpHeaders []string

		burpHeaders, err := input.ParseBurpRequest(burpFilePath)
		if err != nil {
			hqgologger.Fatal("failed parsing burp request file!", hqgologger.WithString("file", burpFilePath), hqgologger.WithError(err))
		}

		h = append(h, burpHeaders...)
	}

	if len(cookies) > 0 {
		h = append(h, "Cookie: "+strings.Join(cookies, "; "))
	}

	h = append(h, headers...)

	var crawlDuration time.Duration

	if viper.GetString("optimization.crawl_duration") != "" {
		var err error

		crawlDuration, err = time.ParseDuration(viper.GetString("optimization.crawl_duration"))
		if err != nil {
			hqgologger.Fatal("failed parsing crawl duration!", hqgologger.WithString("value", viper.GetString("optimization.crawl_duration")), hqgologger.WithError(err))
		}
	}

	crawler, err := xcrawl3r.New(
		xcrawl3r.WithDomains(domains),
		xcrawl3r.WithIncludeSubdomains(includeSubdomains),
		xcrawl3r.WithDelay(time.Duration(viper.GetInt("request.delay"))*time.Second),
		xcrawl3r.WithRandomDelay(time.Duration(viper.GetInt("request.random_delay"))*time.Second),
		xcrawl3r.WithTimeout(time.Duration(viper.GetInt("request.timeout"))*time.Second),
		xcrawl3r.WithRetries(viper.GetInt("request.retries")),
		xcrawl3r.WithRequestsPerMinute(viper.GetInt("request.requests_per_minute")),
		xcrawl3r.WithRandomUserAgent(randomAgent),
		xcrawl3r.WithHeaders(h...),
		xcrawl3r.WithProxies(append(viper.GetStringSlice("proxies"), proxies...)...),
		xcrawl3r.WithDepth(viper.GetInt("optimization.depth")),
		xcrawl3r.WithParallelism(viper.GetInt("optimization.parallelism")),
		xcrawl3r.WithMaxPages(viper.GetInt("optimization.max_pages")),
		xcrawl3r.WithCrawlDuration(crawlDuration),
		xcrawl3r.WithIgnoreQuery(ignoreQuery),
		xcrawl3r.WithShapeDedup(shapeDedup),
		xcrawl3r.WithHeadless(headless),
		xcrawl3r.WithHeadlessTimeout(time.Duration(headlessTimeout)*time.Second),
		xcrawl3r.WithDebug(debug),
	)
	if err != nil {
		hqgologger.Fatal("failed creating crawler!", hqgologger.WithError(err))
	}

	defer crawler.Close()

	if headless {
		if err = crawler.Validate(); err != nil {
			hqgologger.Error("headless rendering unavailable, continuing without it!", hqgologger.WithError(err))
		}
	}

	resultsChan := make(chan xcrawl3r.Result)

	wg := &sync.WaitGroup{}

	for range c {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for URL := range urlsChan {
				for result := range crawler.Crawl(ctx, URL) {
					select {
					case resultsChan <- result:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for result := range resultsChan {
		switch result.Type {
		case xcrawl3r.ResultError:
			if verbose {
				hqgologger.Error("failed to find results!", hqgologger.WithError(result.Error))
			}
		case xcrawl3r.ResultWarning:
			hqgologger.Warn(result.Value)
		case xcrawl3r.ResultURL:
			for _, output := range outputs {
				if err := writer.Write(output, result); err != nil {
					hqgologger.Error("failed to write result!", hqgologger.WithError(err))
				}
			}
		}
	}

	hqgologger.Print("", hqgologger.WithoutTimestamp(), hqgologger.WithoutLabel())
}
