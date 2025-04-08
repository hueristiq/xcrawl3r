# xcrawl3r

![made with go](https://img.shields.io/badge/made%20with-Go-1E90FF.svg) [![go report card](https://goreportcard.com/badge/github.com/hueristiq/xcrawl3r)](https://goreportcard.com/report/github.com/hueristiq/xcrawl3r) [![release](https://img.shields.io/github/release/hueristiq/xcrawl3r?style=flat&color=1E90FF)](https://github.com/hueristiq/xcrawl3r/releases) [![open issues](https://img.shields.io/github/issues-raw/hueristiq/xcrawl3r.svg?style=flat&color=1E90FF)](https://github.com/hueristiq/xcrawl3r/issues?q=is:issue+is:open) [![closed issues](https://img.shields.io/github/issues-closed-raw/hueristiq/xcrawl3r.svg?style=flat&color=1E90FF)](https://github.com/hueristiq/xcrawl3r/issues?q=is:issue+is:closed) [![license](https://img.shields.io/badge/license-MIT-gray.svg?color=1E90FF)](https://github.com/hueristiq/xcrawl3r/blob/master/LICENSE) ![maintenance](https://img.shields.io/badge/maintained%3F-yes-1E90FF.svg) [![contribution](https://img.shields.io/badge/contributions-welcome-1E90FF.svg)](https://github.com/hueristiq/xcrawl3r/blob/master/CONTRIBUTING.md)

`xcrawl3r` is a command-line utility designed to recursively spider webpages for URLs. It works by actively traversing websites - following links embedded in webpages, parsing resources like sitemaps and robots.txt files, and even processing local files - to uncover every URL.

Unlike [`xurlfind3r`](https://github.com/hueristiq/xurlfind3r/) that doesn't interact directly with the target, `xcrawl3r` interacts directly with the target by spidering its pages in real time. This active approach allows it to discover URLs that may be hidden or unindexed, providing a complete picture of the website’s navigational flow and content distribution. This makes `xcrawl3r` a powerful tool for security researchers, IT professionals, and anyone looking to gain insights into the URLs associated with websites.

## Resources

- [Features](#features)
- [Installation](#installation)
	- [Install release binaries (Without Go Installed)](#install-release-binaries-without-go-installed)
	- [Install source (With Go Installed)](#install-source-with-go-installed)
		- [`go install ...`](#go-install)
		- [`go build ...` the development version](#go-build--the-development-version)
	- [Install on Docker (With Docker Installed)](#install-on-docker-with-docker-installed)
- [Usage](#usage)
- [Contributing](#contributing)
- [Licensing](#licensing)

## Features

- Recursively spiders webpages for URLs
- Extract URLs from files (inluding sitemaps & `robots.txt`)
- Cross-Platform (Windows, Linux & macOS)

## Installation

### Install release binaries (Without Go Installed)

Visit the [releases page](https://github.com/hueristiq/xcrawl3r/releases) and find the appropriate archive for your operating system and architecture. Download the archive from your browser or copy its URL and retrieve it with `wget` or `curl`:

- ...with `wget`:

	```bash
	wget https://github.com/hueristiq/xcrawl3r/releases/download/v<version>/xcrawl3r-<version>-linux-amd64.tar.gz
	```

- ...or, with `curl`:

	```bash
	curl -OL https://github.com/hueristiq/xcrawl3r/releases/download/v<version>/xcrawl3r-<version>-linux-amd64.tar.gz
	```

...then, extract the binary:

```bash
tar xf xcrawl3r-<version>-linux-amd64.tar.gz
```

> [!TIP]
> The above steps, download and extract, can be combined into a single step with this onliner
> 
> ```bash
> curl -sL https://github.com/hueristiq/xcrawl3r/releases/download/v<version>/xcrawl3r-<version>-linux-amd64.tar.gz | tar -xzv
> ```

> [!NOTE]
> On Windows systems, you should be able to double-click the zip archive to extract the `xcrawl3r` executable.

...move the `xcrawl3r` binary to somewhere in your `PATH`. For example, on GNU/Linux and OS X systems:

```bash
sudo mv xcrawl3r /usr/local/bin/
```

> [!NOTE]
> Windows users can follow [How to: Add Tool Locations to the PATH Environment Variable](https://msdn.microsoft.com/en-us/library/office/ee537574(v=office.14).aspx) in order to add `xcrawl3r` to their `PATH`.

### Install source (With Go Installed)

Before you install from source, you need to make sure that Go is installed on your system. You can install Go by following the official instructions for your operating system. For this, we will assume that Go is already installed.

#### `go install ...`

```bash
go install -v github.com/hueristiq/xcrawl3r/cmd/xcrawl3r@latest
```

#### `go build ...` the development version

- Clone the repository

	```bash
	git clone https://github.com/hueristiq/xcrawl3r.git 
	```

- Build the utility

	```bash
	cd xcrawl3r/cmd/xcrawl3r && \
	go build .
	```

- Move the `xcrawl3r` binary to somewhere in your `PATH`. For example, on GNU/Linux and OS X systems:

	```bash
	sudo mv xcrawl3r /usr/local/bin/
	```

	Windows users can follow [How to: Add Tool Locations to the PATH Environment Variable](https://msdn.microsoft.com/en-us/library/office/ee537574(v=office.14).aspx) in order to add `xcrawl3r` to their `PATH`.


> [!CAUTION]
> While the development version is a good way to take a peek at `xcrawl3r`'s latest features before they get released, be aware that it may have bugs. Officially released versions will generally be more stable.

### Install on Docker (With Docker Installed)

To install `xcrawl3r` on docker:

- Pull the docker image using:

    ```bash
    docker pull hueristiq/xcrawl3r:latest
    ```

- Run `xcrawl3r` using the image:

    ```bash
    docker run --rm hueristiq/xcrawl3r:latest -h
    ```

## Usage

To start using `xcrawl3r`, open your terminal and run the following command for a list of options:

```bash
xcrawl3r -h
```

Here's what the help message looks like:

```text

                             _ _____
__  _____ _ __ __ ___      _| |___ / _ __
\ \/ / __| '__/ _` \ \ /\ / / | |_ \| '__|
 >  < (__| | | (_| |\ V  V /| |___) | |
/_/\_\___|_|  \__,_| \_/\_/ |_|____/|_|
                                    v0.2.0

USAGE:
 xcrawl3r [OPTIONS]

CONFIGURATION:
 -c, --configuration string       (default: $HOME/.config/xcrawl3r/config.yaml)

INPUT:
 -u, --url string[]               target URL
 -l, --list string                target URLs file path

 For multiple URLs, use comma(,) separated value with `--url`,
 specify multiple `--url`, load from file with `--list` or load from stdin.

SCOPE:
 -d, --domain string[]            match domain(s)  URLs

 For multiple domains, use comma(,) separated value with `--domain`
 or specify multiple `--domain`.

     --include-subdomains bool    with domain(s), match subdomains' URLs

REQUEST:
     --delay int                  delay between each request in seconds
 -H, --header string[]            header to include in 'header:value' format

 For multiple headers, use comma(,) separated value with `--header`
 or specify multiple `--header`.

     --timeout int                time to wait for request in seconds (default: 10)

PROXY:
 -p, --proxy string[]             Proxy (e.g: http://127.0.0.1:8080)

 For multiple proxies use comma(,) separated value with `--proxy`
 or specify multiple `--proxy`.

OPTIMIZATION:
     --depth int                  maximum depth to crawl, `0` for infinite (default: 1)
 -C, --concurrency int            number of concurrent inputs to process (default: 5)
 -P, --parallelism int            number of concurrent fetchers to use (default: 5)

DEBUG:
     --debug bool                 enable debug mode

OUTPUT:
     --jsonl bool                 output in JSONL(ines)
 -o, --output string              output write file path
 -m, --monochrome bool            stdout in monochrome
 -s, --silent bool                stdout in silent mode
 -v, --verbose bool               stdout in verbose mode

```

## Contributing

Contributions are welcome and encouraged! Feel free to submit [Pull Requests](https://github.com/hueristiq/xcrawl3r/pulls) or report [Issues](https://github.com/hueristiq/xcrawl3r/issues). For more details, check out the [contribution guidelines](https://github.com/hueristiq/xcrawl3r/blob/master/CONTRIBUTING.md).

A big thank you to all the [contributors](https://github.com/hueristiq/xcrawl3r/graphs/contributors) for your ongoing support!

![contributors](https://contrib.rocks/image?repo=hueristiq/xcrawl3r&max=500)

## Licensing

This package is licensed under the [MIT license](https://opensource.org/license/mit). You are free to use, modify, and distribute it, as long as you follow the terms of the license. You can find the full license text in the repository - [Full MIT license text](https://github.com/hueristiq/xcrawl3r/blob/master/LICENSE).