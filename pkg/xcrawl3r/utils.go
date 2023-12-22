package xcrawl3r

import (
	"strings"

	"github.com/hueristiq/hqgourl"
)

func decode(source string) (decodedSource string) {
	replacer := strings.NewReplacer(
		`\u002f`, "/",
		`\u0026`, "&",
	)

	decodedSource = replacer.Replace(source)

	return
}

func (crawler *Crawler) fixURL(parsedURL *hqgourl.URL, URL string) (fixedURL string) {
	// decode
	// this ....
	if strings.HasPrefix(URL, "http") {
		// `http://google.com` OR `https://google.com`
		fixedURL = URL
	} else if strings.HasPrefix(URL, "//") {
		// `//google.com/example.php`
		fixedURL = parsedURL.Scheme + ":" + URL
	} else if !strings.HasPrefix(URL, "//") {
		if strings.HasPrefix(URL, "/") {
			// `/?thread=10`
			fixedURL = parsedURL.Scheme + "://" + parsedURL.Host + URL
		} else {
			if strings.HasPrefix(URL, ".") {
				if strings.HasPrefix(URL, "..") {
					// ./style.css
					fixedURL = parsedURL.Scheme + "://" + parsedURL.Host + URL[2:]
				} else {
					// ../style.css
					fixedURL = parsedURL.Scheme + "://" + parsedURL.Host + URL[1:]
				}
			} else {
				// `console/test.php`
				fixedURL = parsedURL.Scheme + "://" + parsedURL.Host + "/" + URL
			}
		}
	}

	return
}

func (crawler *Crawler) IsInScope(URL string) (isInScope bool) {
	parsedURL, err := hqgourl.Parse(URL)
	if err != nil {
		return
	}

	if crawler.IncludeSubdomains {
		isInScope = parsedURL.Domain == crawler.Domain || strings.HasSuffix(parsedURL.Domain, "."+crawler.Domain)
	} else {
		isInScope = parsedURL.Domain == crawler.Domain || parsedURL.Domain == "www."+crawler.Domain
	}

	return
}
