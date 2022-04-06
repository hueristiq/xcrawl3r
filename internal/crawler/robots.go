package crawler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

func (crawler *Crawler) ParseRobots() {
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", crawler.URL.Scheme, crawler.URL.Host)

	res, err := http.Get(robotsURL)
	if err != nil {
		return
	}

	if res.StatusCode == 200 {
		if _, exists := foundURLs.Load(robotsURL); !exists {
			if err := crawler.record(robotsURL); err != nil {
				return
			}

			foundURLs.Store(robotsURL, struct{}{})
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return
		}

		lines := strings.Split(string(body), "\n")

		var re = regexp.MustCompile(".*llow: ")

		for _, line := range lines {
			if strings.Contains(line, "llow: ") {
				URL := re.ReplaceAllString(line, "")

				URL = fmt.Sprintf("%s://%s%s", crawler.URL.Scheme, crawler.URL.Host, URL)

				crawler.PCollector.Visit(URL)
			}
		}

	}
}
