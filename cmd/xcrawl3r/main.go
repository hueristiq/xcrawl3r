package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
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
	"github.com/spf13/viper"
)

var (
	configurationFilePath string
	URLs                  []string
	URLsListFilePath      string
	domains               []string
	includeSubdomains     bool
	delay                 int
	headers               []string
	timeout               int
	proxies               []string
	depth                 int
	concurrency           int
	parallelism           int
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
	pflag.StringSliceVarP(&URLs, "url", "u", []string{}, "")
	pflag.StringVarP(&URLsListFilePath, "list", "l", "", "")
	pflag.StringSliceVarP(&domains, "domain", "d", []string{}, "")
	pflag.BoolVar(&includeSubdomains, "include-subdomains", false, "")
	pflag.IntVar(&delay, "delay", configuration.DefaultConfiguration.Request.Delay, "")
	pflag.StringSliceVarP(&headers, "header", "H", []string{}, "")
	pflag.IntVar(&timeout, "timeout", configuration.DefaultConfiguration.Request.Timeout, "")
	pflag.StringSliceVarP(&proxies, "proxy", "p", []string{}, "")
	pflag.IntVar(&depth, "depth", configuration.DefaultConfiguration.Optimization.Depth, "")
	pflag.IntVarP(&concurrency, "concurrency", "C", configuration.DefaultConfiguration.Optimization.Concurrency, "")
	pflag.IntVarP(&parallelism, "parallelism", "P", configuration.DefaultConfiguration.Optimization.Parallelism, "")
	pflag.BoolVar(&debug, "debug", false, "")
	pflag.BoolVar(&outputInJSONL, "jsonl", false, "")
	pflag.StringVarP(&outputFilePath, "output", "o", "", "")
	pflag.BoolVarP(&monochrome, "monochrome", "m", false, "")
	pflag.BoolVar(&silent, "silent", false, "")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "")

	pflag.Usage = func() {
		hqgologger.Info().Label("").Msg(configuration.BANNER(au))

		h := "USAGE:\n"
		h += fmt.Sprintf(" %s [OPTIONS]\n", configuration.NAME)

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

		h += "\nREQUEST:\n"
		h += "     --delay int                  delay between each request in seconds\n"
		h += " -H, --header string[]            header to include in 'header:value' format\n"

		h += "\n For multiple headers, use comma(,) separated value with `--header`\n"
		h += " or specify multiple `--header`.\n\n"

		h += fmt.Sprintf("     --timeout int                time to wait for request in seconds (default: %d)\n", configuration.DefaultConfiguration.Request.Timeout)

		h += "\nPROXY:\n"
		h += " -p, --proxy string[]             Proxy (e.g: http://127.0.0.1:8080)\n"

		h += "\n For multiple proxies use comma(,) separated value with `--proxy`\n"
		h += " or specify multiple `--proxy`.\n"

		h += "\nOPTIMIZATION:\n"
		h += fmt.Sprintf("     --depth int                  maximum depth to crawl, `0` for infinite (default: %d)\n", configuration.DefaultConfiguration.Optimization.Depth)
		h += fmt.Sprintf(" -C, --concurrency int            number of concurrent inputs to process (default: %d)\n", configuration.DefaultConfiguration.Optimization.Concurrency)
		h += fmt.Sprintf(" -P, --parallelism int            number of concurrent fetchers to use (default: %d)\n", configuration.DefaultConfiguration.Optimization.Parallelism)

		h += "\nDEBUG:\n"
		h += "     --debug bool                 enable debug mode\n"

		h += "\nOUTPUT:\n"
		h += "     --jsonl bool                 output in JSONL(ines)\n"
		h += " -o, --output string              output write file path\n"
		h += " -m, --monochrome bool            stdout in monochrome\n"
		h += " -s, --silent bool                stdout in silent mode\n"
		h += " -v, --verbose bool               stdout in verbose mode\n"

		hqgologger.Info().Label("").Msg(h)
		hqgologger.Print().Msg("")
	}

	pflag.Parse()

	if err := configuration.CreateOrUpdate(configurationFilePath); err != nil {
		hqgologger.Fatal().Msg(err.Error())
	}

	viper.SetConfigFile(configurationFilePath)
	viper.AutomaticEnv()
	viper.SetEnvPrefix(strings.ToUpper(configuration.NAME))
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		hqgologger.Fatal().Msg(err.Error())
	}

	if err := viper.BindPFlag("request.delay", pflag.Lookup("delay")); err != nil {
		hqgologger.Fatal().Msg(err.Error())
	}

	if err := viper.BindPFlag("request.timeout", pflag.Lookup("timeout")); err != nil {
		hqgologger.Fatal().Msg(err.Error())
	}

	if err := viper.BindPFlag("optimization.depth", pflag.Lookup("depth")); err != nil {
		hqgologger.Fatal().Msg(err.Error())
	}

	if err := viper.BindPFlag("optimization.concurrency", pflag.Lookup("concurrency")); err != nil {
		hqgologger.Fatal().Msg(err.Error())
	}

	if err := viper.BindPFlag("optimization.parallelism", pflag.Lookup("parallelism")); err != nil {
		hqgologger.Fatal().Msg(err.Error())
	}

	hqgologger.DefaultLogger.SetFormatter(formatter.NewConsoleFormatter(&formatter.ConsoleFormatterConfiguration{
		Colorize: !monochrome,
	}))

	if silent {
		hqgologger.DefaultLogger.SetLevel(levels.LevelSilent)
	}

	if verbose {
		hqgologger.DefaultLogger.SetLevel(levels.LevelDebug)
	}

	au = aurora.New(aurora.WithColors(!monochrome))
}

func main() {
	hqgologger.Info().Label("").Msg(configuration.BANNER(au))

	c := viper.GetInt("optimization.concurrency")

	URLsChan := make(chan string, c)

	go func() {
		defer close(URLsChan)

		if len(URLs) > 0 {
			for _, URL := range URLs {
				URLsChan <- URL
			}
		}

		if URLsListFilePath != "" {
			file, err := os.Open(URLsListFilePath)
			if err != nil {
				hqgologger.Error().Msg(err.Error())
			}

			scanner := bufio.NewScanner(file)

			for scanner.Scan() {
				URL := scanner.Text()

				if URL != "" {
					URLsChan <- URL
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
					URLsChan <- URL
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

	var file *os.File

	if outputFilePath != "" {
		var err error

		file, err = writer.CreateFile(outputFilePath)
		if err != nil {
			hqgologger.Error().Msg(err.Error())
		}

		outputs = append(outputs, file)
	}

	cfg := &xcrawl3r.Configuration{
		Domains:           domains,
		IncludeSubdomains: includeSubdomains,
		Delay:             viper.GetInt("request.delay"),
		Headers:           append(viper.GetStringSlice("request.headers"), headers...),
		Timeout:           viper.GetInt("request.timeout"),
		Proxies:           append(viper.GetStringSlice("proxies"), proxies...),
		Depth:             viper.GetInt("optimization.depth"),
		Parallelism:       viper.GetInt("optimization.parallelism"),
		Debug:             debug,
	}

	crawler, err := xcrawl3r.New(cfg)
	if err != nil {
		hqgologger.Fatal().Msg(err.Error())
	}

	wg := &sync.WaitGroup{}

	for range c {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for URL := range URLsChan {
				results := crawler.Crawl(URL)

				for result := range results {
					for _, output := range outputs {
						switch result.Type {
						case xcrawl3r.ResultError:
							if verbose {
								hqgologger.Error().Msg(result.Error.Error())
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

	if file != nil {
		file.Close()
	}

	hqgologger.Print().Msg("")
}
