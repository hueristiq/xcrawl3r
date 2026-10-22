package configuration

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"
)

type Request struct {
	Delay              int      `mapstructure:"delay"                yaml:"delay"`
	RandomDelay        int      `mapstructure:"random_delay"         yaml:"random_delay"`
	Headers            []string `mapstructure:"headers"              yaml:"headers"`
	Cookies            []string `mapstructure:"cookies"              yaml:"cookies"`
	Timeout            int      `mapstructure:"timeout"              yaml:"timeout"`
	Retries            int      `mapstructure:"retries"              yaml:"retries"`
	RequestsPerMinute  int      `mapstructure:"requests_per_minute"  yaml:"requests_per_minute"`
	RandomAgent        bool     `mapstructure:"random_agent"         yaml:"random_agent"`
	InsecureSkipVerify bool     `mapstructure:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}

type Optimization struct {
	Depth         int    `mapstructure:"depth"          yaml:"depth"`
	Concurrency   int    `mapstructure:"concurrency"    yaml:"concurrency"`
	Parallelism   int    `mapstructure:"parallelism"    yaml:"parallelism"`
	MaxPages      int    `mapstructure:"max_pages"      yaml:"max_pages"`
	CrawlDuration string `mapstructure:"crawl_duration" yaml:"crawl_duration"`
	IgnoreQuery   bool   `mapstructure:"ignore_query"   yaml:"ignore_query"`
	ShapeDedup    bool   `mapstructure:"shape_dedup"    yaml:"shape_dedup"`
}

type Render struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
	Timeout int  `mapstructure:"timeout" yaml:"timeout"`
	Settle  int  `mapstructure:"settle"  yaml:"settle"`
}

type Configuration struct {
	Version      string       `mapstructure:"version"      yaml:"version"`
	Request      Request      `mapstructure:"request"      yaml:"request"`
	Proxies      []string     `mapstructure:"proxies"      yaml:"proxies"`
	Optimization Optimization `mapstructure:"optimization" yaml:"optimization"`
	Render       Render       `mapstructure:"render"       yaml:"render"`
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

	file, err = os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return
	}

	tempPath := file.Name()

	defer func() {
		if err != nil {
			_ = file.Close()
			_ = os.Remove(tempPath)
		}
	}()

	if err = file.Chmod(0o600); err != nil {
		return
	}

	enc := yaml.NewEncoder(file)

	enc.SetIndent(indentation)

	if err = enc.Encode(cfg); err != nil {
		return
	}

	if err = enc.Close(); err != nil {
		return
	}

	if err = file.Close(); err != nil {
		return
	}

	if err = os.Rename(tempPath, path); err != nil {
		return
	}

	return
}

const Name = "xcrawl3r"

var Version = "v0.0.0-branch-00000000"

var Banner = fmt.Sprintf(`
                                           _ _____
              __  _____ _ __ __ ___      _| |___ / _ __
              \ \/ / __| '__/ _`+"`"+` \ \ /\ / / | |_ \| '__|
               >  < (__| | | (_| |\ V  V /| |___) | |
              /_/\_\___|_|  \__,_| \_/\_/ |_|____/|_| %s

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
		Timeout:            10,
		Retries:            3,
		InsecureSkipVerify: false,
	},
	Proxies: []string{},
	Optimization: Optimization{
		Depth:       2,
		Concurrency: 5,
		Parallelism: 5,
	},
	Render: Render{
		Timeout: 10,
		Settle:  1,
	},
}

var defaultKeys = []string{
	"request.delay",
	"request.random_delay",
	"request.timeout",
	"request.retries",
	"request.requests_per_minute",
	"request.random_agent",
	"request.insecure_skip_verify",
	"optimization.depth",
	"optimization.concurrency",
	"optimization.parallelism",
	"optimization.max_pages",
	"optimization.crawl_duration",
	"optimization.ignore_query",
	"optimization.shape_dedup",
	"render.enabled",
	"render.timeout",
	"render.settle",
}

func CreateOrUpdate(path string) (err error) {
	var cfg Configuration

	var present map[string]bool

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
		cfg, present, err = readWithPresence(path)
		if err != nil {
			return
		}

		if cfg.Version != Version || hasMissingDefaults(present) {
			applyDefaults(&cfg, present)

			cfg.Version = Version

			if err = cfg.Write(path); err != nil {
				return
			}
		}
	}

	return
}

func hasMissingDefaults(present map[string]bool) (missing bool) {
	for _, key := range defaultKeys {
		if !present[key] {
			return true
		}
	}

	return
}

func Read(path string) (cfg Configuration, err error) {
	cfg, _, err = readWithPresence(path)

	return
}

func applyDefaults(cfg *Configuration, present map[string]bool) {
	applyDefaultValues(reflect.ValueOf(cfg).Elem(), reflect.ValueOf(DefaultConfiguration), "", present)
}

func applyDefaultValues(dst, defaults reflect.Value, prefix string, present map[string]bool) {
	typ := dst.Type()

	for index := range typ.NumField() {
		field := typ.Field(index)
		key := field.Tag.Get("mapstructure")

		if key == "" {
			continue
		}

		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if !present[fullKey] {
			dst.Field(index).Set(defaults.Field(index))

			continue
		}

		if field.Type.Kind() == reflect.Struct {
			applyDefaultValues(dst.Field(index), defaults.Field(index), fullKey, present)
		}
	}
}

func readWithPresence(path string) (cfg Configuration, present map[string]bool, err error) {
	file, err := os.Open(path) //nolint:gosec // path is the user-specified configuration file location.
	if err != nil {
		return
	}

	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	var raw map[string]any

	if err = yaml.NewDecoder(file).Decode(&raw); err != nil {
		if !errors.Is(err, io.EOF) {
			return
		}

		err = nil
		raw = nil
	}

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return
	}

	if err = yaml.NewDecoder(file).Decode(&cfg); err != nil {
		if !errors.Is(err, io.EOF) {
			return
		}

		err = nil
	}

	present = make(map[string]bool)

	collectPresence("", raw, present)

	return
}

func collectPresence(prefix string, raw map[string]any, present map[string]bool) {
	for key, value := range raw {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		present[fullKey] = true

		nested, ok := value.(map[string]any)
		if !ok {
			continue
		}

		collectPresence(fullKey, nested, present)
	}
}
