package sigrawl3r

import (
	"strings"

	"github.com/enenumxela/urlx/pkg/urlx"
)

func fixURL(URL string, site *urlx.URL) (fixedURL string) {
	if strings.HasPrefix(URL, "http") {
		// `http://google.com` OR `https://google.com`
		fixedURL = URL
	} else if strings.HasPrefix(URL, "//") {
		// `//google.com/example.php`
		fixedURL = site.Scheme + ":" + URL
	} else if !strings.HasPrefix(URL, "//") {
		if strings.HasPrefix(URL, "/") {
			// `/?thread=10`
			fixedURL = site.Scheme + "://" + site.Host + URL
		} else {
			if strings.HasPrefix(URL, ".") {
				if strings.HasPrefix(URL, "..") {
					fixedURL = site.Scheme + "://" + site.Host + URL[2:]
				} else {
					fixedURL = site.Scheme + "://" + site.Host + URL[1:]
				}
			} else {
				// `console/test.php`
				fixedURL = site.Scheme + "://" + site.Host + "/" + URL
			}
		}
	}

	return fixedURL
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
