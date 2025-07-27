package configuration

import (
	"fmt"
	"os"
	"path/filepath"

	"dario.cat/mergo"
	hqgohttpheader "github.com/hueristiq/hq-go-http/header"
	hqgologger "github.com/hueristiq/hq-go-logger"
	"github.com/logrusorgru/aurora/v4"
	"gopkg.in/yaml.v3"
)

type Request struct {
	Delay   int      `yaml:"delay"`
	Headers []string `yaml:"headers"`
	Timeout int      `yaml:"timeout"`
}

type Optimization struct {
	Depth       int `yaml:"depth"`
	Concurrency int `yaml:"concurrency"`
	Parallelism int `yaml:"Parallelism"`
}

type Configuration struct {
	Version      string       `yaml:"version"`
	Request      Request      `yaml:"request"`
	Proxies      []string     `yaml:"proxies"`
	Optimization Optimization `yaml:"optimization"`
}

func (cfg *Configuration) Write(path string) (err error) {
	var file *os.File

	directory := filepath.Dir(path)
	identation := 4

	if _, err = os.Stat(directory); os.IsNotExist(err) {
		if directory != "" {
			if err = os.MkdirAll(directory, 0o750); err != nil {
				return
			}
		}
	}

	file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return
	}

	defer file.Close()

	enc := yaml.NewEncoder(file)
	enc.SetIndent(identation)
	err = enc.Encode(&cfg)

	return
}

const (
	NAME    = "xcrawl3r"
	VERSION = "1.0.0"
)

var (
	BANNER = func(au *aurora.Aurora) (banner string) {
		banner = au.Sprintf(
			au.BrightBlue(`
                             _ _____
__  _____ _ __ __ ___      _| |___ / _ __
\ \/ / __| '__/ _`+"`"+` \ \ /\ / / | |_ \| '__|
 >  < (__| | | (_| |\ V  V /| |___) | |
/_/\_\___|_|  \__,_| \_/\_/ |_|____/|_|
                                    %s`).Bold(),
			au.BrightRed("v"+VERSION).Bold().Italic(),
		) + "\n\n"

		return
	}

	UserDotConfigDirectoryPath = func() (userDotConfig string) {
		var err error

		userDotConfig, err = os.UserConfigDir()
		if err != nil {
			hqgologger.Fatal("failed getting `$HOME/.config/`", hqgologger.WithError(err))
		}

		return
	}()

	DefaultConfigurationFilePath = filepath.Join(UserDotConfigDirectoryPath, NAME, "config.yaml")
	DefaultConfiguration         = Configuration{
		Version: VERSION,
		Request: Request{
			Delay: 0,
			Headers: []string{
				fmt.Sprintf("%s: %s v%s (https://github.com/hueristiq/%s)", hqgohttpheader.UserAgent, NAME, VERSION, NAME),
			},
			Timeout: 10,
		},
		Proxies: []string{},
		Optimization: Optimization{
			Depth:       1,
			Concurrency: 5,
			Parallelism: 5,
		},
	}
)

func CreateOrUpdate(path string) (err error) {
	var cfg Configuration

	_, err = os.Stat(path)

	switch {
	case err != nil && os.IsNotExist(err):
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

		if cfg.Version != VERSION {
			if err = mergo.Merge(&cfg, DefaultConfiguration); err != nil {
				return
			}

			cfg.Version = VERSION

			if err = cfg.Write(path); err != nil {
				return
			}
		}
	}

	return
}

func Read(path string) (cfg Configuration, err error) {
	var file *os.File

	file, err = os.Open(path)
	if err != nil {
		return
	}

	defer file.Close()

	if err = yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return
	}

	return
}
