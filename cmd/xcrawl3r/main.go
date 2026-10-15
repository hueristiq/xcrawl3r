package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	hqgologger "github.com/hueristiq/hq-lib-logger-go"
	hqgologgerformatter "github.com/hueristiq/hq-lib-logger-go/formatter"
	hqgologgerlevels "github.com/hueristiq/hq-lib-logger-go/levels"
	hqgourlparser "github.com/hueristiq/hq-lib-url-go/parser"
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
	timeout               int
	retries               int
	requestsPerMinute     int
	randomAgent           bool
	insecureSkipVerify    bool
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
	outputInJSONL         bool
	outputFilePath        string
	outputDirectoryPath   string
	monochrome            bool
	silent                bool
	verbose               bool

	au = aurora.New(aurora.WithColors(true))
)

var parser = hqgourlparser.New(hqgourlparser.WithDefaultScheme("https"))

type crawledResult struct {
	domain string
	result xcrawl3r.Result
}

func rootDomain(rawURL string) (domain string) {
	parsed, err := parser.Parse(rawURL)
	if err != nil {
		return
	}

	return parsed.Key("apex")
}

func init() {
	pflag.StringVarP(&configurationFilePath, "configuration", "c", configuration.DefaultConfigurationFilePath(), "")
	pflag.StringSliceVarP(&urls, "url", "u", []string{}, "")
	pflag.StringVarP(&urlsListFilePath, "list", "l", "", "")
	pflag.StringSliceVarP(&domains, "domain", "d", []string{}, "")
	pflag.BoolVar(&includeSubdomains, "include-subdomains", false, "")
	pflag.IntVar(&delay, "delay", configuration.DefaultConfiguration.Request.Delay, "")
	pflag.IntVar(&randomDelay, "random-delay", configuration.DefaultConfiguration.Request.RandomDelay, "")
	pflag.StringSliceVarP(&headers, "header", "H", []string{}, "")
	pflag.StringSliceVar(&cookies, "cookie", []string{}, "")
	pflag.IntVar(&timeout, "timeout", configuration.DefaultConfiguration.Request.Timeout, "")
	pflag.IntVar(&retries, "retries", configuration.DefaultConfiguration.Request.Retries, "")
	pflag.IntVar(&requestsPerMinute, "requests-per-minute", configuration.DefaultConfiguration.Request.RequestsPerMinute, "")
	pflag.BoolVar(&randomAgent, "random-agent", configuration.DefaultConfiguration.Request.RandomAgent, "")
	pflag.BoolVar(&insecureSkipVerify, "insecure-skip-verify", configuration.DefaultConfiguration.Request.InsecureSkipVerify, "skip TLS certificate verification")
	pflag.StringSliceVar(&proxies, "proxy", []string{}, "")
	pflag.IntVar(&depth, "depth", configuration.DefaultConfiguration.Optimization.Depth, "")
	pflag.IntVarP(&concurrency, "concurrency", "C", configuration.DefaultConfiguration.Optimization.Concurrency, "")
	pflag.IntVarP(&parallelism, "parallelism", "P", configuration.DefaultConfiguration.Optimization.Parallelism, "")
	pflag.IntVar(&maxPages, "max-pages", configuration.DefaultConfiguration.Optimization.MaxPages, "")
	pflag.StringVar(&crawlDuration, "crawl-duration", configuration.DefaultConfiguration.Optimization.CrawlDuration, "")
	pflag.BoolVar(&ignoreQuery, "ignore-query", configuration.DefaultConfiguration.Optimization.IgnoreQuery, "")
	pflag.BoolVar(&shapeDedup, "shape-dedup", configuration.DefaultConfiguration.Optimization.ShapeDedup, "")
	pflag.BoolVar(&headless, "headless", configuration.DefaultConfiguration.Headless.Enabled, "")
	pflag.IntVar(&headlessTimeout, "headless-timeout", configuration.DefaultConfiguration.Headless.Timeout, "")
	pflag.BoolVar(&outputInJSONL, "jsonl", false, "")
	pflag.StringVarP(&outputFilePath, "output", "o", "", "")
	pflag.StringVarP(&outputDirectoryPath, "output-directory", "O", "", "")
	pflag.BoolVarP(&monochrome, "monochrome", "m", false, "")
	pflag.BoolVar(&silent, "silent", false, "")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "")

	pflag.Usage = func() {
		hqgologger.Info(configuration.Banner(au), hqgologger.WithoutTimestamp(), hqgologger.WithoutLabel())

		h := "USAGE:\n"
		h += fmt.Sprintf(" %s [OPTIONS]\n", configuration.Name)

		h += "\nCONFIGURATION:\n"

		defaultConfigurationFilePath := configuration.DefaultConfigurationFilePath()

		if userConfigDirectoryPath, err := configuration.UserConfigDirectoryPath(); err == nil {
			defaultConfigurationFilePath = strings.ReplaceAll(defaultConfigurationFilePath, userConfigDirectoryPath, "$HOME/.config")
		}

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
		h += fmt.Sprintf("     --retries int                maximum attempts per request, `0` for no retries (default: %d)\n", configuration.DefaultConfiguration.Request.Retries)
		h += "     --random-agent bool          use a random User-Agent for each request\n"
		h += "     --insecure-skip-verify bool  skip TLS certificate verification (default: true)\n"
		h += "     --requests-per-minute int    maximum requests per minute, per host (default: 0, no limit)\n"
		h += "     --cookie string[]            cookie to include in 'name=value' format\n"

		h += "\nPROXY:\n"
		h += "     --proxy string[]             Proxy (e.g: http://127.0.0.1:8080)\n"

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

		h += "\nHEADLESS:\n"
		h += "     --headless bool              render pages in a headless browser, requires Chrome/Chromium\n"
		h += "     --headless-timeout int       time to wait for page rendering in seconds (default: 10)\n"

		h += "\nOUTPUT:\n"
		h += "     --jsonl bool                 output in JSONL(ines)\n"
		h += " -o, --output string              output write file path\n"
		h += " -O, --output-directory string    output write directory path\n"
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

	if err := viper.BindPFlag("request.random_agent", pflag.Lookup("random-agent")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "random-agent"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("request.insecure_skip_verify", pflag.Lookup("insecure-skip-verify")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "insecure-skip-verify"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("optimization.ignore_query", pflag.Lookup("ignore-query")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "ignore-query"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("optimization.shape_dedup", pflag.Lookup("shape-dedup")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "shape-dedup"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("headless.enabled", pflag.Lookup("headless")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "headless"), hqgologger.WithError(err))
	}

	if err := viper.BindPFlag("headless.timeout", pflag.Lookup("headless-timeout")); err != nil {
		hqgologger.Fatal("failed binding flag!", hqgologger.WithString("flag", "headless-timeout"), hqgologger.WithError(err))
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
	hqgologger.Info(configuration.Banner(au), hqgologger.WithoutLabel())

	signal.Ignore(syscall.SIGPIPE)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	c := viper.GetInt("optimization.concurrency")
	if c < 1 {
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
			hqgologger.Fatal("failed to create output file!", hqgologger.WithString("file", outputFilePath), hqgologger.WithError(err))
		}

		defer func() {
			_ = file.Close()
		}()

		outputs = append(outputs, file)
	}

	h := viper.GetStringSlice("request.headers")

	if !viper.GetBool("request.random_agent") {
		h = configuration.WithDefaultUserAgent(h)
	}

	allCookies := append(viper.GetStringSlice("request.cookies"), cookies...)

	if len(allCookies) > 0 {
		h = append(h, "Cookie: "+strings.Join(allCookies, "; "))
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
		xcrawl3r.WithRandomUserAgent(viper.GetBool("request.random_agent")),
		xcrawl3r.WithInsecureSkipVerify(viper.GetBool("request.insecure_skip_verify")),
		xcrawl3r.WithHeaders(h...),
		xcrawl3r.WithProxies(append(viper.GetStringSlice("proxies"), proxies...)...),
		xcrawl3r.WithDepth(viper.GetInt("optimization.depth")),
		xcrawl3r.WithParallelism(viper.GetInt("optimization.parallelism")),
		xcrawl3r.WithMaxPages(viper.GetInt("optimization.max_pages")),
		xcrawl3r.WithCrawlDuration(crawlDuration),
		xcrawl3r.WithIgnoreQuery(viper.GetBool("optimization.ignore_query")),
		xcrawl3r.WithShapeDedup(viper.GetBool("optimization.shape_dedup")),
		xcrawl3r.WithHeadless(viper.GetBool("headless.enabled")),
		xcrawl3r.WithHeadlessTimeout(time.Duration(viper.GetInt("headless.timeout"))*time.Second),
	)
	if err != nil {
		hqgologger.Fatal("failed creating crawler!", hqgologger.WithError(err))
	}

	defer crawler.Close()

	if viper.GetBool("headless.enabled") {
		if err = crawler.Validate(); err != nil {
			hqgologger.Error("headless rendering unavailable, continuing without it!", hqgologger.WithError(err))
		}
	}

	resultsChan := make(chan crawledResult)

	wg := &sync.WaitGroup{}

	for range c {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for URL := range urlsChan {
				domain := rootDomain(URL)

				for result := range crawler.Crawl(ctx, URL) {
					select {
					case resultsChan <- crawledResult{
						domain: domain,
						result: result,
					}:
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

	domainFiles := make(map[string]*os.File)

	domainFile := func(domain string) (file *os.File, err error) {
		file, ok := domainFiles[domain]
		if ok {
			return file, nil
		}

		file, err = writer.CreateFile(filepath.Join(outputDirectoryPath, domain))
		if err != nil {
			return nil, fmt.Errorf("creating output file for %s: %w", domain, err)
		}

		domainFiles[domain] = file

		return file, nil
	}

	for crawled := range resultsChan {
		result := crawled.result

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

			if outputFilePath == "" && outputDirectoryPath != "" && crawled.domain != "" {
				file, err := domainFile(crawled.domain)
				if err != nil {
					hqgologger.Fatal("failed to create output file!", hqgologger.WithString("file", filepath.Join(outputDirectoryPath, crawled.domain)), hqgologger.WithError(err))
				}

				if err := writer.Write(file, result); err != nil {
					hqgologger.Error("failed to write result!", hqgologger.WithError(err))
				}

				if err := writer.Flush(file); err != nil {
					hqgologger.Error("failed to flush result!", hqgologger.WithError(err))
				}
			}

			for _, output := range outputs {
				if err := writer.Flush(output); err != nil {
					hqgologger.Error("failed to flush result!", hqgologger.WithError(err))
				}
			}
		}
	}

	for _, file := range domainFiles {
		_ = file.Close()
	}

	hqgologger.Print("", hqgologger.WithoutTimestamp(), hqgologger.WithoutLabel())
}
