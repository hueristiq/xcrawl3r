package configuration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"dario.cat/mergo"
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

// booleanPreserver keeps mergo from treating an explicit `false` in the
// user's configuration as an unset field to overwrite with a default.
type booleanPreserver struct{}

func (booleanPreserver) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ.Kind() == reflect.Bool {
		return func(dst, src reflect.Value) error {
			return nil
		}
	}

	return nil
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

		if cfg.Version != Version || hasMissingDefaults(&cfg) {
			if err = mergo.Merge(&cfg, DefaultConfiguration, mergo.WithTransformers(booleanPreserver{})); err != nil {
				return
			}

			cfg.Version = Version

			if err = cfg.Write(path); err != nil {
				return
			}
		}
	}

	return
}

func hasMissingDefaults(cfg *Configuration) (missing bool) {
	if cfg.Request.Timeout == 0 {
		return true
	}

	if cfg.Request.Retries == 0 {
		return true
	}

	if cfg.Optimization.Depth == 0 {
		return true
	}

	if cfg.Optimization.Concurrency == 0 {
		return true
	}

	if cfg.Optimization.Parallelism == 0 {
		return true
	}

	if cfg.Headless.Timeout == 0 {
		return true
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
