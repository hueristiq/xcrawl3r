package xcrawl3r

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	hqgohttp "github.com/hueristiq/hq-go-http"
	"github.com/hueristiq/hq-go-http/status"
	hqgourl "github.com/hueristiq/hq-go-url"
)

func (crawler *Crawler) robotsParsing(parsedURL *hqgourl.URL) <-chan Result {
	results := make(chan Result)

	go func() {
		defer close(results)

		robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsedURL.Scheme, parsedURL.Host)

		res, err := hqgohttp.Get(robotsURL)
		if err != nil {
			result := Result{
				Type:   ResultError,
				Source: "known:robots",
				Error:  err,
			}

			results <- result

			return
		}

		defer res.Body.Close()

		if res.StatusCode != status.OK {
			result := Result{
				Type:   ResultError,
				Source: "known:robots",
				Error:  errors.New("unexpected status code"),
			}

			results <- result

			return
		}

		result := Result{
			Type:   ResultURL,
			Source: "known:robots",
			Value:  robotsURL,
		}

		results <- result

		body, err := io.ReadAll(res.Body)
		if err != nil {
			result := Result{
				Type:   ResultError,
				Source: "known:robots",
				Error:  err,
			}

			results <- result

			return
		}

		lines := strings.Split(string(body), "\n")

		re := regexp.MustCompile(".*llow: ")

		for _, line := range lines {
			if !strings.Contains(line, "llow: ") {
				continue
			}

			rfURL := re.ReplaceAllString(line, "")

			rfURL = strings.ReplaceAll(rfURL, "*", "")
			rfURL = strings.TrimPrefix(rfURL, "/")
			rfURL = fmt.Sprintf("%s://%s/%s", parsedURL.Scheme, parsedURL.Host, rfURL)

			result := Result{
				Type:   ResultURL,
				Source: "robots",
				Value:  rfURL,
			}

			results <- result

			if err = crawler.PageCollector.Visit(rfURL); err != nil {
				result := Result{
					Type:   ResultError,
					Source: "robots",
					Error:  err,
				}

				results <- result

				continue
			}
		}
	}()

	return results
}
