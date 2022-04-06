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

     _                          _
 ___(_) __ _ _ __ __ ___      _| | ___ _ __
/ __| |/ _` | '__/ _` \ \ /\ / / |/ _ \ '__|
\__ \ | (_| | | | (_| |\ V  V /| |  __/ |
|___/_|\__, |_|  \__,_| \_/\_/ |_|\___|_| v1.2.0
       |___/

USAGE:
  sigrawl3r [OPTIONS]

OPTIONS:
  -debug          debug mode (default: false)
  -delay          delay between requests. (default 5s)
  -depth          maximum limit on the recursion depth of visited URLs. (default 1)
  -iL             urls to crawl (use `iL -` to read from stdin)
  -iS             extend scope to include subdomains (default: false)
  -nC             no color mode
  -oJ             JSON output file
  -s              silent mode: print urls only (default: false)
  -threads        maximum no. of concurrent requests (default 20)
  -timeout        HTTP timeout (default 10s)
  -UA             User Agent to use
  -x              comma separated list of proxies
```

## Installation

#### From Binary

You can download the pre-built binary for your platform from this repository's [releases](https://github.com/signedsecurity/sigrawl3r/releases/) page, extract, then move it to your `$PATH`and you're ready to go.

#### From Source

sigrawl3r requires **go1.14+** to install successfully. Run the following command to get the repo

```bash
GO111MODULE=on go get -u -v github.com/signedsecurity/sigrawl3r/cmd/sigrawl3r
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