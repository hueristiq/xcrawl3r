# X Crawler (`xcrawl3r`)

![made with go](https://img.shields.io/badge/made%20with-Go-1E90FF.svg) [![go report card](https://goreportcard.com/badge/github.com/hueristiq/xcrawl3r)](https://goreportcard.com/report/github.com/hueristiq/xcrawl3r) [![release](https://img.shields.io/github/release/hueristiq/xcrawl3r?style=flat&color=1E90FF)](https://github.com/hueristiq/xcrawl3r/releases) [![open issues](https://img.shields.io/github/issues-raw/hueristiq/xcrawl3r.svg?style=flat&color=1E90FF)](https://github.com/hueristiq/xcrawl3r/issues?q=is:issue+is:open) [![closed issues](https://img.shields.io/github/issues-closed-raw/hueristiq/xcrawl3r.svg?style=flat&color=1E90FF)](https://github.com/hueristiq/xcrawl3r/issues?q=is:issue+is:closed) [![license](https://img.shields.io/badge/license-MIT-gray.svg?color=1E90FF)](https://github.com/hueristiq/xcrawl3r/blob/master/LICENSE) ![maintenance](https://img.shields.io/badge/maintained%3F-yes-1E90FF.svg) [![contribution](https://img.shields.io/badge/contributions-welcome-1E90FF.svg)](https://github.com/hueristiq/xcrawl3r/blob/master/CONTRIBUTING.md)

`xcrawl3r` is a command-line interface (CLI) based utility to recursively crawl webpages. It is designed to systematically browse webpages' URLs and follow links to discover linked webpages' URLs.

## Resources

* [Features](#features)
* [Installation](#installation)
	* [Install release binaries (Without Go Installed)](#install-release-binaries-without-go-installed)
	* [Install source (With Go Installed)](#install-source-with-go-installed)
		* [`go install ...`](#go-install)
		* [`go build ...` the development Version](#go-build--the-development-version)
	* [Install on Docker (With Docker Installed)](#install-on-docker-with-docker-installed)
* [Usage](#usage)
* [Contributing](#contributing)
* [Licensing](#licensing)
* [Credits](#credits)
	* [Contributors](#contributors)
	* [Similar Projects](#similar-projects)

## Features

* Recursively crawls webpages for URLs.
* Parses URLs from files (`.js`, `.json`, `.xml`, `.csv`, `.txt` & `.map`).
* Parses URLs from `robots.txt`.
* Parses URLs from sitemaps.
* Renders pages (including Single Page Applications such as Angular and React).
* Cross-Platform (Windows, Linux & macOS)

## Installation

### Install release binaries (Without Go Installed)

Visit the [releases page](https://github.com/hueristiq/xcrawl3r/releases) and find the appropriate archive for your operating system and architecture. Download the archive from your browser or copy its URL and retrieve it with `wget` or `curl`:

* ...with `wget`:

	```bash
	wget https://github.com/hueristiq/xcrawl3r/releases/download/v<version>/xcrawl3r-<version>-linux-amd64.tar.gz
	```

* ...or, with `curl`:

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

#### `go build ...` the development Version

* Clone the repository

	```bash
	git clone https://github.com/hueristiq/xcrawl3r.git 
	```

* Build the utility

	```bash
	cd xcrawl3r/cmd/xcrawl3r && \
	go build .
	```

* Move the `xcrawl3r` binary to somewhere in your `PATH`. For example, on GNU/Linux and OS X systems:

	```bash
	sudo mv xcrawl3r /usr/local/bin/
	```

	Windows users can follow [How to: Add Tool Locations to the PATH Environment Variable](https://msdn.microsoft.com/en-us/library/office/ee537574(v=office.14).aspx) in order to add `xcrawl3r` to their `PATH`.


> [!CAUTION]
> While the development version is a good way to take a peek at `xcrawl3r`'s latest features before they get released, be aware that it may have bugs. Officially released versions will generally be more stable.

### Install on Docker (With Docker Installed)

To install `xcrawl3r` on docker:

* Pull the docker image using:

    ```bash
    docker pull hueristiq/xcrawl3r:latest
    ```

* Run `xcrawl3r` using the image:

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

INPUT:
 -d, --domain string               domain to match URLs
     --include-subdomains bool     match subdomains' URLs
 -s, --seeds string                seed URLs file (use `-` to get from stdin)
 -u, --url string                  URL to crawl

CONFIGURATION:
     --depth int                   maximum depth to crawl (default 3)
                                      TIP: set it to `0` for infinite recursion
     --headless bool               If true the browser will be displayed while crawling.
 -H, --headers string[]            custom header to include in requests
                                      e.g. -H 'Referer: http://example.com/'
                                      TIP: use multiple flag to set multiple headers
     --proxy string[]              Proxy URL (e.g: http://127.0.0.1:8080)
                                      TIP: use multiple flag to set multiple proxies
     --render bool                 utilize a headless chrome instance to render pages
     --timeout int                 time to wait for request in seconds (default: 10)
     --user-agent string           User Agent to use (default: xcrawl3r v0.2.0 (https://github.com/hueristiq/xcrawl3r))
                                      TIP: use `web` for a random web user-agent,
                                      `mobile` for a random mobile user-agent,
                                       or you can set your specific user-agent.

RATE LIMIT:
 -c, --concurrency int             number of concurrent fetchers to use (default 10)
     --delay int                   delay between each request in seconds
     --max-random-delay int        maximux extra randomized delay added to `--dalay` (default: 1s)
 -p, --parallelism int             number of concurrent URLs to process (default: 10)

OUTPUT:
     --debug bool                  enable debug mode (default: false)
 -m, --monochrome bool             coloring: no colored output mode
 -o, --output string               output file to write found URLs
     --silent bool                 display output URLs only
 -v, --verbose bool                display verbose output

```

## Contributing

We welcome contributions! Feel free to submit [Pull Requests](https://github.com/hueristiq/xcrawl3r/pulls) or report [Issues](https://github.com/hueristiq/xcrawl3r/issues). For more details, check out the [contribution guidelines](https://github.com/hueristiq/xcrawl3r/blob/master/CONTRIBUTING.md).

## Licensing

This utility is licensed under the [MIT license](https://opensource.org/license/mit). You are free to use, modify, and distribute it, as long as you follow the terms of the license. You can find the full license text in the repository - [Full MIT license text](https://github.com/hueristiq/xcrawl3r/blob/master/LICENSE).

## Credits

### Contributors

A huge thanks to all the contributors who have helped make `xcrawl3r` what it is today!

[![contributors](https://contrib.rocks/image?repo=hueristiq/xcrawl3r&max=500)](https://github.com/hueristiq/xcrawl3r/graphs/contributors)

### Similar Projects

If you're interested in more utilities like this, check out:

[gospider](https://github.com/jaeles-project/gospider) ◇ [hakrawler](https://github.com/hakluke/hakrawler) ◇ [katana](https://github.com/projectdiscovery/katana) ◇ [urlgrab](https://github.com/IAmStoxe/urlgrab)