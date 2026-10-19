package configuration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	hqgohttpheader "github.com/hueristiq/hq-lib-http-go/header"
	"gopkg.in/yaml.v3"
)

type Request struct {
	Delay              int      `yaml:"delay"`
	RandomDelay        int      `yaml:"random_delay"`
	Headers            []string `yaml:"headers"`
	Cookies            []string `yaml:"cookies"`
	Timeout            int      `yaml:"timeout"`
	Retries            int      `yaml:"retries"`
	RequestsPerMinute  int      `yaml:"requests_per_minute"`
	RandomAgent        bool     `yaml:"random_agent"`
	InsecureSkipVerify bool     `yaml:"insecure_skip_verify"`
}

type Optimization struct {
	Depth         int    `yaml:"depth"`
	Concurrency   int    `yaml:"concurrency"`
	Parallelism   int    `yaml:"parallelism"`
	MaxPages      int    `yaml:"max_pages"`
	CrawlDuration string `yaml:"crawl_duration"`
	IgnoreQuery   bool   `yaml:"ignore_query"`
	ShapeDedup    bool   `yaml:"shape_dedup"`
}

type Headless struct {
	Enabled bool `yaml:"enabled"`
	Timeout int  `yaml:"timeout"`
}

type Configuration struct {
	Version      string       `yaml:"version"`
	Request      Request      `yaml:"request"`
	Proxies      []string     `yaml:"proxies"`
	Optimization Optimization `yaml:"optimization"`
	Headless     Headless     `yaml:"headless"`
}

func (cfg *Configuration) Write(path string) (err error) {
	directory := filepath.Dir(path)
	indentation := 4

	if _, err = os.Stat(directory); errors.Is(err, os.ErrNotExist) {
		if directory != "" {
			if err = os.MkdirAll(directory, 0o750); err != nil {
				return
			}
		}
	}

	var file *os.File

	file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) //nolint:gosec // path is the user-specified configuration file location.
	if err != nil {
		return
	}

	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	enc := yaml.NewEncoder(file)

	enc.SetIndent(indentation)

	if err = enc.Encode(cfg); err != nil {
		return
	}

	if err = enc.Close(); err != nil {
		return
	}

	return
}

const Name = "xcrawl3r"

var Version = "0.0.0"

var Banner = fmt.Sprintf(`
                                           _ _____
              __  _____ _ __ __ ___      _| |___ / _ __
              \ \/ / __| '__/ _`+"`"+` \ \ /\ / / | |_ \| '__|
               >  < (__| | | (_| |\ V  V /| |___) | |
              /_/\_\___|_|  \__,_| \_/\_/ |_|____/|_| v%s-dev-f711bbd

                 Hueristiq Limited <hueristiq.com>

`, Version)

func UserConfigDirectoryPath() (userConfigDirectoryPath string, err error) {
	userConfigDirectoryPath, err = os.UserConfigDir()
	if err != nil {
		err = fmt.Errorf("failed to resolve user configuration directory: %w", err)

		return
	}

	return
}

func DefaultConfigurationFilePath() (path string) {
	userConfigDirectoryPath, err := UserConfigDirectoryPath()
	if err != nil {
		userConfigDirectoryPath = "."
	}

	path = filepath.Join(userConfigDirectoryPath, Name, "config.yaml")

	return
}

var DefaultConfiguration = Configuration{
	Version: Version,
	Request: Request{
		Headers: []string{
			DefaultUserAgent,
		},
		Timeout:            10,
		Retries:            3,
		InsecureSkipVerify: true,
	},
	Proxies: []string{},
	Optimization: Optimization{
		Depth:       1,
		Concurrency: 5,
		Parallelism: 5,
	},
	Headless: Headless{
		Timeout: 10,
	},
}

// DefaultUserAgent is the built-in User-Agent header value, applied only
// when no User-Agent is configured.
var DefaultUserAgent = fmt.Sprintf("%s: %s v%s (https://github.com/hueristiq/%s)", hqgohttpheader.UserAgent, Name, Version, Name)

// WithDefaultUserAgent appends the built-in User-Agent header to headers when
// none of the entries is a User-Agent header.
func WithDefaultUserAgent(headers []string) (withUserAgent []string) {
	for _, header := range headers {
		key, _, found := strings.Cut(header, ":")
		if found && strings.EqualFold(strings.TrimSpace(key), hqgohttpheader.UserAgent.String()) {
			return headers
		}
	}

	return append(headers, DefaultUserAgent)
}

func CreateOrUpdate(path string) (err error) {
	var cfg Configuration

	_, err = os.Stat(path)

	switch {
	case errors.Is(err, os.ErrNotExist):
		cfg = DefaultConfiguration

		if err = cfg.Write(path); err != nil {
			return
		}
	case err != nil:
		return
	default:
		cfg, err = Read(path)
		if err != nil {
			return
		}

		raw, rawErr := readRaw(path)
		if rawErr != nil {
			return rawErr
		}

		if cfg.Version != Version || fillMissingDefaults(&cfg, raw) {
			cfg.Version = Version

			if err = cfg.Write(path); err != nil {
				return
			}
		}
	}

	return
}

func Read(path string) (cfg Configuration, err error) {
	file, err := os.Open(path) //nolint:gosec // path is the user-specified configuration file location.
	if err != nil {
		return
	}

	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	if err = yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return
	}

	return
}

func readRaw(path string) (raw map[string]any, err error) {
	file, err := os.Open(path) //nolint:gosec // path is the user-specified configuration file location.
	if err != nil {
		return
	}

	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	raw = make(map[string]any)

	if err = yaml.NewDecoder(file).Decode(&raw); err != nil {
		return
	}

	return
}

func rawHas(raw map[string]any, path ...string) (has bool) {
	var node any = raw

	for _, key := range path {
		mapping, ok := node.(map[string]any)
		if !ok {
			return false
		}

		node, ok = mapping[key]
		if !ok {
			return false
		}
	}

	return true
}

func fillMissingDefaults(cfg *Configuration, raw map[string]any) (changed bool) {
	changed = fillRequestDefaults(cfg, raw)

	if !rawHas(raw, "proxies") {
		cfg.Proxies = DefaultConfiguration.Proxies
		changed = true
	}

	if fillOptimizationDefaults(cfg, raw) {
		changed = true
	}

	if fillHeadlessDefaults(cfg, raw) {
		changed = true
	}

	return
}

func fillRequestDefaults(cfg *Configuration, raw map[string]any) (changed bool) {
	request, ok := raw["request"].(map[string]any)
	if !ok {
		if !rawHas(raw, "request") {
			cfg.Request = DefaultConfiguration.Request
			changed = true
		}

		return
	}

	if !rawHas(request, "delay") {
		cfg.Request.Delay = DefaultConfiguration.Request.Delay
		changed = true
	}

	if !rawHas(request, "random_delay") {
		cfg.Request.RandomDelay = DefaultConfiguration.Request.RandomDelay
		changed = true
	}

	if !rawHas(request, "headers") {
		cfg.Request.Headers = DefaultConfiguration.Request.Headers
		changed = true
	}

	if !rawHas(request, "cookies") {
		cfg.Request.Cookies = DefaultConfiguration.Request.Cookies
		changed = true
	}

	if !rawHas(request, "timeout") {
		cfg.Request.Timeout = DefaultConfiguration.Request.Timeout
		changed = true
	}

	if !rawHas(request, "retries") {
		cfg.Request.Retries = DefaultConfiguration.Request.Retries
		changed = true
	}

	if !rawHas(request, "requests_per_minute") {
		cfg.Request.RequestsPerMinute = DefaultConfiguration.Request.RequestsPerMinute
		changed = true
	}

	if !rawHas(request, "random_agent") {
		cfg.Request.RandomAgent = DefaultConfiguration.Request.RandomAgent
		changed = true
	}

	if !rawHas(request, "insecure_skip_verify") {
		cfg.Request.InsecureSkipVerify = DefaultConfiguration.Request.InsecureSkipVerify
		changed = true
	}

	return
}

func fillOptimizationDefaults(cfg *Configuration, raw map[string]any) (changed bool) {
	optimization, ok := raw["optimization"].(map[string]any)
	if !ok {
		if !rawHas(raw, "optimization") {
			cfg.Optimization = DefaultConfiguration.Optimization
			changed = true
		}

		return
	}

	if !rawHas(optimization, "depth") {
		cfg.Optimization.Depth = DefaultConfiguration.Optimization.Depth
		changed = true
	}

	if !rawHas(optimization, "concurrency") {
		cfg.Optimization.Concurrency = DefaultConfiguration.Optimization.Concurrency
		changed = true
	}

	if !rawHas(optimization, "parallelism") {
		cfg.Optimization.Parallelism = DefaultConfiguration.Optimization.Parallelism
		changed = true
	}

	if !rawHas(optimization, "max_pages") {
		cfg.Optimization.MaxPages = DefaultConfiguration.Optimization.MaxPages
		changed = true
	}

	if !rawHas(optimization, "crawl_duration") {
		cfg.Optimization.CrawlDuration = DefaultConfiguration.Optimization.CrawlDuration
		changed = true
	}

	if !rawHas(optimization, "ignore_query") {
		cfg.Optimization.IgnoreQuery = DefaultConfiguration.Optimization.IgnoreQuery
		changed = true
	}

	if !rawHas(optimization, "shape_dedup") {
		cfg.Optimization.ShapeDedup = DefaultConfiguration.Optimization.ShapeDedup
		changed = true
	}

	return
}

func fillHeadlessDefaults(cfg *Configuration, raw map[string]any) (changed bool) {
	headless, ok := raw["headless"].(map[string]any)
	if !ok {
		if !rawHas(raw, "headless") {
			cfg.Headless = DefaultConfiguration.Headless
			changed = true
		}

		return
	}

	if !rawHas(headless, "enabled") {
		cfg.Headless.Enabled = DefaultConfiguration.Headless.Enabled
		changed = true
	}

	if !rawHas(headless, "timeout") {
		cfg.Headless.Timeout = DefaultConfiguration.Headless.Timeout
		changed = true
	}

	return
}
