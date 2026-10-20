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
	render                bool
	renderTimeout         int
	outputInJSONL         bool
	outputFilePath        string
	outputDirectoryPath   string
	monochrome            bool
	silent                bool
	verbose               bool
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
	pflag.StringSliceVar(&urls, "url", []string{}, "")
	pflag.StringVarP(&urlsListFilePath, "list", "l", "", "")
	pflag.StringSliceVarP(&domains, "domain", "d", []string{}, "")
	pflag.BoolVar(&includeSubdomains, "include-subdomains", false, "")
	pflag.StringSliceVar(&proxies, "proxy", []string{}, "")
	pflag.IntVar(&delay, "delay", 0, "")
	pflag.IntVar(&randomDelay, "random-delay", 0, "")
	pflag.StringSliceVarP(&headers, "header", "H", []string{}, "")
	pflag.BoolVar(&randomAgent, "random-agent", false, "")
	pflag.StringSliceVar(&cookies, "cookie", []string{}, "")
	pflag.BoolVar(&insecureSkipVerify, "insecure-skip-verify", false, "skip TLS certificate verification")
	pflag.IntVar(&timeout, "timeout", 0, "")
	pflag.IntVar(&retries, "retries", 0, "")
	pflag.IntVar(&depth, "depth", 0, "")
	pflag.IntVar(&maxPages, "max-pages", 0, "")
	pflag.IntVarP(&concurrency, "concurrency", "C", 0, "")
	pflag.IntVarP(&parallelism, "parallelism", "P", 0, "")
	pflag.IntVar(&requestsPerMinute, "requests-per-minute", 0, "")
	pflag.StringVar(&crawlDuration, "crawl-duration", "", "")
	pflag.BoolVar(&ignoreQuery, "ignore-query", false, "")
	pflag.BoolVar(&shapeDedup, "shape-dedup", false, "")
	pflag.BoolVar(&render, "render", false, "")
	pflag.IntVar(&renderTimeout, "render-timeout", 0, "")
	pflag.BoolVar(&outputInJSONL, "jsonl", false, "")
	pflag.StringVarP(&outputFilePath, "output", "o", "", "")
	pflag.StringVarP(&outputDirectoryPath, "output-directory", "O", "", "")
	pflag.BoolVarP(&monochrome, "monochrome", "m", false, "")
	pflag.BoolVarP(&silent, "silent", "s", false, "")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "")

	pflag.Usage = func() {
		hqgologger.Info(configuration.Banner, hqgologger.WithoutTimestamp(), hqgologger.WithoutLabel())

		h := "USAGE:\n"
		h += fmt.Sprintf(" %s [OPTIONS]\n", configuration.Name)

		h += "\nCONFIGURATION:\n"

		defaultConfigurationFilePath := configuration.DefaultConfigurationFilePath()

		if userConfigDirectoryPath, err := configuration.UserConfigDirectoryPath(); err == nil {
			defaultConfigurationFilePath = strings.ReplaceAll(defaultConfigurationFilePath, userConfigDirectoryPath, "$HOME/.config")
		}

		h += fmt.Sprintf(" -c, --configuration string       (default: %v)\n", defaultConfigurationFilePath)

		h += "\nINPUT:\n"
		h += "     --url string[]               target URL\n"
		h += " -l, --list string                target URLs file path\n"

		h += "\n For multiple URLs, use comma(,) separated value with `--url`,\n"
		h += " specify multiple `--url`, load from file with `--list` or load from stdin.\n"

		h += "\nSCOPE:\n"
		h += " -d, --domain string[]            domain to match URLs for\n"

		h += "\n For multiple domains, use comma(,) separated value with `--domain`\n"
		h += " or specify multiple `--domain`."
		h += "\n Without `--domain`, only URLs on the target's host are matched.\n"

		h += "\n     --include-subdomains bool    match subdomain's URLs\n"

		h += "\nPROXY:\n"
		h += "     --proxy string[]             proxy (e.g: http://127.0.0.1:8080)\n"

		h += "\n For multiple proxies, use comma(,) separated value with `--proxy`\n"
		h += " or specify multiple `--proxy`.\n"

		h += "\nOPTIMIZATION:\n"
		h += "     --delay int                  delay between each request in seconds\n"
		h += "     --random-delay int           random delay between each request in seconds\n"
		h += " -H, --header string[]            header to include in 'header:value' format\n"

		h += "\n For multiple headers, use comma(,) separated value with `--header`\n"
		h += " or specify multiple `--header`.\n"

		h += "\n     --random-agent bool          use a random User-Agent for each request\n"
		h += "     --cookie string[]            cookie to include in 'name=value' format\n"
		h += "     --insecure-skip-verify bool  skip TLS certificate verification\n"
		h += "     --timeout int                time to wait for request in seconds\n"
		h += "     --retries int                maximum attempts per request, `0` for no retries\n"

		h += "     --depth int                  maximum depth to crawl, `0` for infinite\n"
		h += "     --max-pages int              maximum pages to crawl per target, `0` for no limit\n"
		h += " -C, --concurrency int            number of concurrent inputs to process\n"
		h += " -P, --parallelism int            number of concurrent fetchers to use\n"
		h += "     --requests-per-minute int    maximum requests per minute, per host, `0` for no limit\n"
		h += "     --crawl-duration duration    maximum duration per target crawl (e.g: 10s, 5m), `0` for no limit\n"
		h += "     --ignore-query bool          deduplicate URLs ignoring the query string\n"
		h += "     --shape-dedup bool           deduplicate URLs by shape (numbers, UUIDs, dates normalized)\n"
		h += "     --render bool                render pages in a headless browser, requires Chrome/Chromium\n"
		h += "     --render-timeout int         time to wait for page rendering in seconds\n"

		h += "\nOUTPUT:\n"
		h += "     --jsonl bool                 output in JSONL(ines)\n"
		h += " -o, --output string              output write file path\n"
		h += " -O, --output-directory string    output write directory path\n"
		h += " -m, --monochrome bool            stdout in monochrome\n"
		h += " -s, --silent bool                stdout in silent mode\n"
		h += " -v, --verbose bool               stdout in verbose mode\n"

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

	hqgologger.DefaultLogger.SetFormatter(
		hqgologgerformatter.NewConsoleFormatter(&hqgologgerformatter.ConsoleFormatterConfiguration{
			Colorize:     !monochrome,
			IncludeLabel: true,
		}),
	)

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
}

func main() {
	hqgologger.Info(configuration.Banner, hqgologger.WithoutLabel())

	var cfg *configuration.Configuration

	if err := viper.Unmarshal(&cfg); err != nil {
		hqgologger.Fatal("failed unmarshalling Configuration!", hqgologger.WithError(err))
	}

	if urlsListFilePath != "" {
		file, err := os.Open(urlsListFilePath) //nolint:gosec // path is the user-specified URLs file location.
		if err != nil {
			hqgologger.Fatal("failed opening input file", hqgologger.WithError(err))
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			urls = append(urls, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			hqgologger.Fatal("failed reading input file!", hqgologger.WithError(err))
		}

		if err := file.Close(); err != nil {
			hqgologger.Fatal("failed closing input file!", hqgologger.WithError(err))
		}
	}

	if input.HasStdin() {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			urls = append(urls, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			hqgologger.Fatal("failed reading stdin!", hqgologger.WithError(err))
		}
	}

	validatedDomains := make([]string, 0, len(domains))

	for _, domain := range domains {
		domain = strings.TrimSpace(domain)

		if domain == "" {
			continue
		}

		if err := input.ValidateDomain(domain); err != nil {
			hqgologger.Fatal("invalid domain!", hqgologger.WithString("domain", domain), hqgologger.WithError(err))
		}

		validatedDomains = append(validatedDomains, domain)
	}

	domains = validatedDomains

	validatedURLs := make([]string, 0, len(urls))

	for _, URL := range urls {
		URL = strings.TrimSpace(URL)

		if URL == "" {
			continue
		}

		if _, err := parser.Parse(URL); err != nil {
			hqgologger.Fatal("invalid url!", hqgologger.WithString("url", URL), hqgologger.WithError(err))
		}

		validatedURLs = append(validatedURLs, URL)
	}

	urls = validatedURLs

	if len(urls) == 0 {
		hqgologger.Fatal("no targets provided! specify at least one with --url, --list or stdin.")
	}

	writer := output.NewWriter()

	if outputInJSONL {
		writer.SetFormatToJSONL()
	}

	signal.Ignore(syscall.SIGPIPE)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	defer stop()

	if !pflag.Lookup("delay").Changed {
		delay = cfg.Request.Delay
	}

	if !pflag.Lookup("random-delay").Changed {
		randomDelay = cfg.Request.RandomDelay
	}

	if !pflag.Lookup("timeout").Changed {
		timeout = cfg.Request.Timeout
	}

	if !pflag.Lookup("retries").Changed {
		retries = cfg.Request.Retries
	}

	if !pflag.Lookup("requests-per-minute").Changed {
		requestsPerMinute = cfg.Request.RequestsPerMinute
	}

	if !pflag.Lookup("random-agent").Changed {
		randomAgent = cfg.Request.RandomAgent
	}

	if !pflag.Lookup("insecure-skip-verify").Changed {
		insecureSkipVerify = cfg.Request.InsecureSkipVerify
	}

	if !pflag.Lookup("depth").Changed {
		depth = cfg.Optimization.Depth
	}

	if !pflag.Lookup("concurrency").Changed {
		concurrency = cfg.Optimization.Concurrency
	}

	if !pflag.Lookup("parallelism").Changed {
		parallelism = cfg.Optimization.Parallelism
	}

	if !pflag.Lookup("max-pages").Changed {
		maxPages = cfg.Optimization.MaxPages
	}

	if !pflag.Lookup("crawl-duration").Changed {
		crawlDuration = cfg.Optimization.CrawlDuration
	}

	if !pflag.Lookup("ignore-query").Changed {
		ignoreQuery = cfg.Optimization.IgnoreQuery
	}

	if !pflag.Lookup("shape-dedup").Changed {
		shapeDedup = cfg.Optimization.ShapeDedup
	}

	if !pflag.Lookup("render").Changed {
		render = cfg.Render.Enabled
	}

	if !pflag.Lookup("render-timeout").Changed {
		renderTimeout = cfg.Render.Timeout
	}

	headers = append(cfg.Request.Headers, headers...)
	cookies = append(cfg.Request.Cookies, cookies...)
	proxies = append(cfg.Proxies, proxies...)

	if len(cookies) > 0 {
		headers = append(headers, "Cookie: "+strings.Join(cookies, "; "))
	}

	var parsedCrawlDuration time.Duration

	if crawlDuration != "" {
		var err error

		parsedCrawlDuration, err = time.ParseDuration(crawlDuration)
		if err != nil {
			hqgologger.Fatal("failed parsing crawl duration!", hqgologger.WithString("value", crawlDuration), hqgologger.WithError(err))
		}
	}

	crawler, err := xcrawl3r.New(&xcrawl3r.Configuration{
		Client: &xcrawl3r.ClientConfiguration{
			UserAgent: fmt.Sprintf("%s %s", configuration.Name, configuration.Version),
		},
		Domains:            domains,
		IncludeSubdomains:  includeSubdomains,
		Delay:              time.Duration(delay) * time.Second,
		RandomDelay:        time.Duration(randomDelay) * time.Second,
		Timeout:            time.Duration(timeout) * time.Second,
		Retries:            retries,
		RequestsPerMinute:  requestsPerMinute,
		RandomUserAgent:    randomAgent,
		InsecureSkipVerify: insecureSkipVerify,
		Headers:            headers,
		Proxies:            proxies,
		Depth:              depth,
		Parallelism:        parallelism,
		MaxPages:           maxPages,
		CrawlDuration:      parsedCrawlDuration,
		IgnoreQuery:        ignoreQuery,
		ShapeDedup:         shapeDedup,
		Render:             render,
		RenderTimeout:      time.Duration(renderTimeout) * time.Second,
	})
	if err != nil {
		hqgologger.Fatal("failed creating crawler!", hqgologger.WithError(err))
	}

	defer crawler.Close()

	if render {
		if err = crawler.Validate(); err != nil {
			hqgologger.Error("JS rendering unavailable, continuing without it!", hqgologger.WithError(err))
		}
	}

	outputs := []io.Writer{
		os.Stdout,
	}

	var file *os.File

	if outputFilePath != "" {
		file, err = writer.CreateFile(outputFilePath)
		if err != nil {
			hqgologger.Fatal("failed to create output file!", hqgologger.WithString("file", outputFilePath), hqgologger.WithError(err))
		}

		outputs = append(outputs, file)
	}

	if concurrency < 1 {
		concurrency = 1
	}

	urlsChan := make(chan string, concurrency)

	go func() {
		defer close(urlsChan)

		for i := range urls {
			select {
			case urlsChan <- urls[i]:
			case <-ctx.Done():
				return
			}
		}
	}()

	resultsChan := make(chan crawledResult)

	wg := &sync.WaitGroup{}

	for range concurrency {
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
				hqgologger.Error("failed to find results!", hqgologger.WithString("source", result.Source), hqgologger.WithError(result.Error))
			}
		case xcrawl3r.ResultWarning:
			hqgologger.Warn(result.Value)
		case xcrawl3r.ResultURL:
			domain := rootDomain(result.Value)
			if domain == "" {
				domain = crawled.domain
			}

			for _, output := range outputs {
				write := writer.Write

				if output == os.Stdout && !silent {
					write = writer.WriteWithTechnique
				}

				if err := write(output, domain, result); err != nil {
					if output != os.Stdout {
						_ = writer.Flush(os.Stdout)

						hqgologger.Fatal("failed to write result to output file!", hqgologger.WithError(err))
					}

					outputs = dropOutput(outputs, output)

					hqgologger.Error("failed to write result to stdout; dropping stdout output!", hqgologger.WithError(err))

					if len(outputs) == 0 {
						stop()
					}
				}
			}

			if outputFilePath == "" && outputDirectoryPath != "" && domain != "" {
				domainOutput, err := domainFile(domain)
				if err != nil {
					hqgologger.Fatal("failed to create output file!", hqgologger.WithString("file", filepath.Join(outputDirectoryPath, domain)), hqgologger.WithError(err))
				}

				if err := writer.Write(domainOutput, domain, result); err != nil {
					hqgologger.Fatal("failed to write result to output file!", hqgologger.WithError(err))
				}

				if err := writer.Flush(domainOutput); err != nil {
					hqgologger.Fatal("failed to flush output file!", hqgologger.WithError(err))
				}
			}

			for _, output := range outputs {
				if err := writer.Flush(output); err != nil {
					if output != os.Stdout {
						_ = writer.Flush(os.Stdout)

						hqgologger.Fatal("failed to flush output file!", hqgologger.WithError(err))
					}

					outputs = dropOutput(outputs, output)

					hqgologger.Error("failed to flush stdout; dropping stdout output!", hqgologger.WithError(err))

					if len(outputs) == 0 {
						stop()
					}
				}
			}
		}
	}

	for _, domainOutput := range domainFiles {
		if err := domainOutput.Close(); err != nil {
			hqgologger.Fatal("failed closing output file!", hqgologger.WithError(err))
		}
	}

	if file != nil {
		if err := file.Close(); err != nil {
			hqgologger.Fatal("failed closing output file!", hqgologger.WithError(err))
		}
	}

	hqgologger.Print("", hqgologger.WithoutTimestamp(), hqgologger.WithoutLabel())
}

func dropOutput(outputs []io.Writer, dropped io.Writer) (remaining []io.Writer) {
	remaining = make([]io.Writer, 0, len(outputs))

	for _, output := range outputs {
		if output != dropped {
			remaining = append(remaining, output)
		}
	}

	return
}
