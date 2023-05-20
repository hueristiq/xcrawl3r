# xcrawl3r

![made with go](https://img.shields.io/badge/made%20with-Go-0000FF.svg) [![release](https://img.shields.io/github/release/hueristiq/xcrawl3r?style=flat&color=0000FF)](https://github.com/hueristiq/xcrawl3r/releases) [![license](https://img.shields.io/badge/license-MIT-gray.svg?color=0000FF)](https://github.com/hueristiq/xcrawl3r/blob/master/LICENSE) ![maintenance](https://img.shields.io/badge/maintained%3F-yes-0000FF.svg) [![open issues](https://img.shields.io/github/issues-raw/hueristiq/xcrawl3r.svg?style=flat&color=0000FF)](https://github.com/hueristiq/xcrawl3r/issues?q=is:issue+is:open) [![closed issues](https://img.shields.io/github/issues-closed-raw/hueristiq/xcrawl3r.svg?style=flat&color=0000FF)](https://github.com/hueristiq/xcrawl3r/issues?q=is:issue+is:closed) [![contribution](https://img.shields.io/badge/contributions-welcome-0000FF.svg)](https://github.com/hueristiq/xcrawl3r/blob/master/CONTRIBUTING.md)

`xcrawl3r` is a command-line interface (CLI) utility to recursively crawl webpages i.e systematically browse webpages' URLs and follow links to discover linked webpages' URLs.

## Resources

* [Features](#features)
* [Installation](#installation)
	* [Install release binaries](#install-release-binaries)
	* [Install source](#install-sources)
		* [`go install ...`](#go-install)
		* [`go build ...` the development Version](#go-build--the-development-version)
* [Usage](#usage)
* [Contribution](#contribution)
* [Licensing](#licensing)

## Features

* Recursively crawls webpages for URLs.
* Parses files for URLs. (`.js`, `.json`, `.xml`, `.csv`, `.txt` & `.map`) 
* Parses `robots.txt` for URLs.
* Parses sitemaps for URLs.
* Customizable Parallelism

## Installation

### Install release binaries

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

> **TIP:** The above steps, download and extract, can be combined into a single step with this onliner
> 
> ```bash
> curl -sL https://github.com/hueristiq/xcrawl3r/releases/download/v<version>/xcrawl3r-<version>-linux-amd64.tar.gz | tar -xzv
> ```

**NOTE:** On Windows systems, you should be able to double-click the zip archive to extract the `xcrawl3r` executable.

...move the `xcrawl3r` binary to somewhere in your `PATH`. For example, on GNU/Linux and OS X systems:

```bash
sudo mv xcrawl3r /usr/local/bin/
```

**NOTE:** Windows users can follow [How to: Add Tool Locations to the PATH Environment Variable](https://msdn.microsoft.com/en-us/library/office/ee537574(v=office.14).aspx) in order to add `xcrawl3r` to their `PATH`.

### Install source

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

	**NOTE:** Windows users can follow [How to: Add Tool Locations to the PATH Environment Variable](https://msdn.microsoft.com/en-us/library/office/ee537574(v=office.14).aspx) in order to add `xcrawl3r` to their `PATH`.


**NOTE:** While the development version is a good way to take a peek at `xcrawl3r`'s latest features before they get released, be aware that it may have bugs. Officially released versions will generally be more stable.

## Usage

To display help message for `xcrawl3r` use the `-h` flag:

```bash
xcrawl3r -h
```

help message:

```text
                             _ _____      
__  _____ _ __ __ ___      _| |___ / _ __ 
\ \/ / __| '__/ _` \ \ /\ / / | |_ \| '__|
 >  < (__| | | (_| |\ V  V /| |___) | |   
/_/\_\___|_|  \__,_| \_/\_/ |_|____/|_| v0.0.0

A CLI utility to recursively crawl webpages.

USAGE:
  xcrawl3r [OPTIONS]

INPUT:
  -d, --domain string              domain to match URLs
      --include-subdomains bool    match subdomains' URLs
  -s, --seeds string               seed URLs file (use `-` to get from stdin)
  -u, --url string                 URL to crawl

CONFIGURATION:
      --depth int                  maximum depth to crawl (default 3)
                                       TIP: set it to `0` for infinite recursion
      --timeout int               time to wait for request in seconds (default: 10)
  -H, --headers string[]          custom header to include in requests
                                       e.g. -H 'Referer: http://example.com/'
                                       TIP: use multiple flag to set multiple headers
      --user-agent string         User Agent to use (default: web)
                                       TIP: use `web` for a random web user-agent,
                                       `mobile` for a random mobile user-agent,
                                        or you can set your specific user-agent.
      --proxy string[]            Proxy URL (e.g: http://127.0.0.1:8080)
                                       TIP: use multiple flag to set multiple proxies

RATE LIMIT:
  -c, --concurrency int           number of concurrent fetchers to use (default 10)
  -p, --parallelism int           number of concurrent URLs to process (default: 10)
      --delay int                 delay between each request in seconds
      --max-random-delay int      maximux extra randomized delay added to `--dalay` (default: 1s)

OUTPUT:
      --debug bool                 enable debug mode (default: false)
  -m, --monochrome bool            coloring: no colored output mode
  -o, --output string              output file to write found URLs
  -v, --verbosity string           debug, info, warning, error, fatal or silent (default: debug)
```

## Contributing

[Issues](https://github.com/hueristiq/xcrawl3r/issues) and [Pull Requests](https://github.com/hueristiq/xcrawl3r/pulls) are welcome! Check out the [contribution guidelines](./CONTRIBUTING.md).

## Licensing

This utility is distributed under the [MIT license](./LICENSE).