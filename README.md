# sigrawl3r

[![release](https://img.shields.io/github/release/signedsecurity/sigrawl3r?style=flat&color=0040ff)](https://github.com/signedsecurity/sigrawl3r/releases) [![maintenance](https://img.shields.io/badge/maintained%3F-yes-0040ff.svg)](https://github.com/signedsecurity/sigrawl3r) [![open issues](https://img.shields.io/github/issues-raw/signedsecurity/sigrawl3r.svg?style=flat&color=0040ff)](https://github.com/signedsecurity/sigrawl3r/issues?q=is:issue+is:open) [![closed issues](https://img.shields.io/github/issues-closed-raw/signedsecurity/sigrawl3r.svg?style=flat&color=0040ff)](https://github.com/signedsecurity/sigrawl3r/issues?q=is:issue+is:closed) [![license](https://img.shields.io/badge/license-MIT-gray.svg?colorB=0040FF)](https://github.com/signedsecurity/sigrawl3r/blob/master/LICENSE) [![twitter](https://img.shields.io/badge/twitter-@signedsecurity-0040ff.svg)](https://twitter.com/signedsecurity)

A fast web crawler.

## Resources

* [Features](#features)
* [Usage](#usage)
* [Installation](#installation)
	* [From Binary](#from-binary)
	* [From source](#from-source)
	* [From github](#from-github)
* [Contribution](#contribution)

## Features

* Parses sitemap for URLs.
* Parses robots.txt for URLs.
* Extracts URLs from documents *(including js, json, xml, csv, txt e.t.c)*.
* Supports javaScript rendering *(including Single Page Applications such as Angular, React, e.t.c)*.
* Supports customizable parallelism.

## Usage

```text
$ sigrawl3r -h

     _                          _ _____      
 ___(_) __ _ _ __ __ ___      _| |___ / _ __ 
/ __| |/ _` | '__/ _` \ \ /\ / / | |_ \| '__|
\__ \ | (_| | | | (_| |\ V  V /| |___) | |   
|___/_|\__, |_|  \__,_| \_/\_/ |_|____/|_| v1.0.0
       |___/

USAGE:
  sigrawl3r [OPTIONS]

OPTIONS:
  -c, --concurrency          Maximum concurrent requests for matching domains (default: 5)
      --debug                Enable debug mode (default: false)
  -d, --depth                Maximum recursion depth on visited URLs. (default: 1)
      --headless             If true the browser will be displayed while crawling
                                 Note: Requires '-r, --render' flag
                                 Note: Usage to show browser: '--headless=false' (default true)
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

## Installation

#### From Binary

You can download the pre-built binary for your platform from this repository's [releases](https://github.com/signedsecurity/sigrawl3r/releases/) page, extract, then move it to your `$PATH`and you're ready to go.

#### From Source

sigrawl3r requires **go1.17+** to install successfully. Run the following command to get the repo

```bash
go install github.com/signedsecurity/signed-suite-cli/cmd/sigcli@latest
```

#### From Github

```bash
git clone https://github.com/signedsecurity/sigrawl3r.git && \
cd sigrawl3r/cmd/sigrawl3r/ && \
go build . && \
mv sigrawl3r /usr/local/bin/ && \
sigrawl3r -h
```

## Contribution

[Issues](https://github.com/signedsecurity/sigrawl3r/issues) and [Pull Requests](https://github.com/signedsecurity/sigrawl3r/pulls) are welcome! 