package hqcrawl3r

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	hqurl "github.com/hueristiq/hqgoutils/url"
)

func (crawler *Crawler) fixURL(URL string) (fixedURL string) {
	// decode
	// this ....
	if strings.HasPrefix(URL, "http") {
		// `http://google.com` OR `https://google.com`
		fixedURL = URL
	} else if strings.HasPrefix(URL, "//") {
		// `//google.com/example.php`
		fixedURL = crawler.URL.Scheme + ":" + URL
	} else if !strings.HasPrefix(URL, "//") {
		if strings.HasPrefix(URL, "/") {
			// `/?thread=10`
			fixedURL = crawler.URL.Scheme + "://" + crawler.URL.Host + URL
		} else {
			if strings.HasPrefix(URL, ".") {
				if strings.HasPrefix(URL, "..") {
					// ./style.css
					fixedURL = crawler.URL.Scheme + "://" + crawler.URL.Host + URL[2:]
				} else {
					// ../style.css
					fixedURL = crawler.URL.Scheme + "://" + crawler.URL.Host + URL[1:]
				}
			} else {
				// `console/test.php`
				fixedURL = crawler.URL.Scheme + "://" + crawler.URL.Host + "/" + URL
			}
		}
	}

	return
}

func decode(URL string) string {
	// In case json encoded chars
	replacer := strings.NewReplacer(
		`\u002f`, "/",
		`\u0026`, "&",
	)

	URL = replacer.Replace(strings.ToLower(URL))

	return URL
}

func (crawler *Crawler) record(URL string) (err error) {
	URL = decode(URL)

	parsedURL, err := hqurl.Parse(URL)
	if err != nil {
		return
	}

	print := false

	if crawler.Configuration.IncludeSubdomains {
		escapedHost := strings.ReplaceAll(crawler.URL.Host, ".", "\\.")
		print, _ = regexp.MatchString(".*(\\.|\\/\\/)"+escapedHost+"((#|\\/|\\?).*)?", URL)
	} else {
		print = parsedURL.Host == crawler.URL.Host || parsedURL.Host == "www."+crawler.URL.Host
	}

	if print {
		fmt.Fprintln(os.Stdout, parsedURL.String())
	}

	return
}
