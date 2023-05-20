package xcrawl3r

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	hqurl "github.com/hueristiq/hqgoutils/url"
)

func (crawler *Crawler) robotsParsing(parsedURL *hqurl.URL) (URLsChannel chan URL) {
	URLsChannel = make(chan URL)

	go func() {
		defer close(URLsChannel)

		robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)

		res, err := http.Get(robotsURL) //nolint:gosec // Works!
		if err != nil {
			return
		}

		defer res.Body.Close()

		if res.StatusCode == 200 {
			URLsChannel <- URL{Source: "known", Value: robotsURL}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return
			}

			lines := strings.Split(string(body), "\n")

			re := regexp.MustCompile(".*llow: ")

			for _, line := range lines {
				if strings.Contains(line, "llow: ") {
					rfURL := re.ReplaceAllString(line, "")
					rfURL = fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, rfURL)

					URLsChannel <- URL{Source: "robots", Value: rfURL}

					if err = crawler.PageCollector.Visit(rfURL); err != nil {
						continue
					}
				}
			}
		}
	}()

	return
}
