package hqcrawl3r

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	hqurl "github.com/hueristiq/hqgoutils/url"
)

var lfRegex = regexp.MustCompile(`(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\?|#][^"|']{0,}|)))(?:"|')`)

func decode(source string) (decodedSource string) {
	replacer := strings.NewReplacer(
		`\u002f`, "/",
		`\u0026`, "&",
	)

	decodedSource = replacer.Replace(source)

	return
}

func extractLinks(source string) (links []string, err error) {
	links = []string{}
	links = append(links, lfRegex.FindAllString(source, -1)...)

	return
}

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

func (crawler *Crawler) record(URL string) (err error) {
	parsedURL, err := hqurl.Parse(URL)
	if err != nil {
		return
	}

	print := false

	if crawler.Options.IncludeSubdomains {
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
