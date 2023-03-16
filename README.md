# hqcrawl3r

[![release](https://img.shields.io/github/release/hueristiq/hqcrawl3r?style=flat&color=0040ff)](https://github.com/hueristiq/hqcrawl3r/releases) ![maintenance](https://img.shields.io/badge/maintained%3F-yes-0040ff.svg) [![open issues](https://img.shields.io/github/issues-raw/hueristiq/hqcrawl3r.svg?style=flat&color=0040ff)](https://github.com/hueristiq/hqcrawl3r/issues?q=is:issue+is:open) [![closed issues](https://img.shields.io/github/issues-closed-raw/hueristiq/hqcrawl3r.svg?style=flat&color=0040ff)](https://github.com/hueristiq/hqcrawl3r/issues?q=is:issue+is:closed) [![license](https://img.shields.io/badge/license-MIT-gray.svg?colorB=0040FF)](https://github.com/hueristiq/hqcrawl3r/blob/master/LICENSE)

`hqcrawl3r` is a command-line interface (CLI) utility to recursively crawl webpages. With this utility, you can systematically browse webpages' URLs and follow links to discover linked webpages' URLs.

## Resources

* [Features](#features)
* [Installation](#installation)
	* [Install release binaries](#install-release-binaries)
	* [Install source](#install-sources)
		* [`go install ...`](#go-install)
		* [`go build ...` the development Version](#go-build--the-development-version)
* [Usage](#usage)
* [Contributing](#contributing)
* [Licensing](#licensing)

## Features

* Parses sitemap for URLs
* Parses `robots.txt` for URLs
* Recursively crawls webpages for URLs.
* Parses documents (`js`, `json`, `xml`, `csv`, `txt`, e.t.c) for URLs.

## Installation

### Install release binaries

Visit the [releases page](https://github.com/hueristiq/hqcrawl3r/releases) and find the appropriate archive for your operating system and architecture. Download the archive from your browser or copy its URL and retrieve it with `wget` or `curl`:

* ...with `wget`:

	```bash
	wget https://github.com/hueristiq/hqcrawl3r/releases/download/v<version>/hqcrawl3r-<version>-linux-amd64.tar.gz
	```

* ...or, with `curl`:

	```bash
	curl -OL https://github.com/hueristiq/hqcrawl3r/releases/download/v<version>/hqcrawl3r-<version>-linux-amd64.tar.gz
	```

...then, extract the binary:

```bash
tar xf hqcrawl3r-<version>-linux-amd64.tar.gz
```

> **TIP:** The above steps, download and extract, can be combined into a single step with this onliner
> 
> ```bash
> curl -sL https://github.com/hueristiq/hqcrawl3r/releases/download/v<version>/hqcrawl3r-<version>-linux-amd64.tar.gz | tar -xzv
> ```

**NOTE:** On Windows systems, you should be able to double-click the zip archive to extract the `hqcrawl3r` executable.

...move the `hqcrawl3r` binary to somewhere in your `PATH`. For example, on GNU/Linux and OS X systems:

```bash
sudo mv hqcrawl3r /usr/local/bin/
```

**NOTE:** Windows users can follow [How to: Add Tool Locations to the PATH Environment Variable](https://msdn.microsoft.com/en-us/library/office/ee537574(v=office.14).aspx) in order to add `hqcrawl3r` to their `PATH`.

### Install source

Before you install from source, you need to make sure that Go is installed on your system. You can install Go by following the official instructions for your operating system. For this, we will assume that Go is already installed.

#### `go install ...`

```bash
go install -v github.com/hueristiq/hqcrawl3r/cmd/hqcrawl3r@latest
```

#### `go build ...` the development Version

* Clone the Repository

	```bash
	git clone https://github.com/hueristiq/hqcrawl3r.git 
	```

* Build the Program

	```bash
	cd hqcrawl3r/cmd/hqcrawl3r/ && \
	go build .
	```

* Move the `hqcrawl3r` binary to somewhere in your `PATH`. For example, on GNU/Linux and OS X systems:

	```bash
	sudo mv hqcrawl3r /usr/local/bin/
	```

	**NOTE:** Windows users can follow [How to: Add Tool Locations to the PATH Environment Variable](https://msdn.microsoft.com/en-us/library/office/ee537574(v=office.14).aspx) in order to add `hqcrawl3r` to their `PATH`.


**NOTE:** While the development version is a good way to take a peek at `hqcrawl3r`'s latest features before they get released, be aware that it may have bugs. Officially released versions will generally be more stable.

## Usage

Display help message:

```bash
hqcrawl3r -h
```

```text
 _                                   _ _____
| |__   __ _  ___ _ __ __ ___      _| |___ / _ __
| '_ \ / _` |/ __| '__/ _` \ \ /\ / / | |_ \| '__|
| | | | (_| | (__| | | (_| |\ V  V /| |___) | |
|_| |_|\__, |\___|_|  \__,_| \_/\_/ |_|____/|_|
          |_|                            v0.0.0

[> A CLI utility to recursively crawl webpages. <]

USAGE:
  hqcrawl3r [OPTIONS]

OPTIONS:
  -c, --concurrency          Maximum concurrent requests for matching domains (default: 5)
      --cookie               Cookie to use (testA=a; testB=b)
      --debug                Enable debug mode (default: false)
  -d, --depth                Maximum recursion depth on visited URLs. (default: 1)
      --headless             If true the browser will be displayed while crawling
                                 Note: Requires '-r, --render' flag
                                 Note: Usage to show browser: '--headless=false' (default true)
  -H, --headers              Custom headers separated by two semi-colons.
                                 E.g. -h 'Cookie: foo=bar;;Referer: http://example.com/'
      --include-subs         Extend scope to include subdomains (default: false)
      --no-color             Enable no color mode (default: false)
  -p, --proxy                Proxy URL (e.g: http://127.0.0.1:8080)
  -R, --random-delay         Maximum random delay between requests (default: 2s)
  -r, --render               Render javascript.
  -s, --silent               Enable silent mode: output urls only (default: false)
  -t, --threads              Number of threads (Run URLs in parallel) (default: 20)
      --timeout              Request timeout (second) (default: 10)
  -u, --url                  URL to crawl
  -U, --urls                 URLs to crawl
      --user-agent           User Agent to use (default: web)
                                 `web` for a random web user-agent
                                 `mobile` for a random mobile user-agent
                                 or you can set your special user-agent
```

## Contributing

[Issues](https://github.com/hueristiq/hqcrawl3r/issues) and [Pull Requests](https://github.com/hueristiq/hqcrawl3r/pulls) are welcome! Check out the [contribution guidelines.](./CONTRIBUTING.md)

## Licensing

The tool is licensed under the [MIT license](./LICENSE)