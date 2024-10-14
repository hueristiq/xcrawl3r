package xcrawl3r

import (
	"mime"
	"path"
	"strings"

	"github.com/gocolly/colly/v2"
	hqgourl "github.com/hueristiq/hq-go-url"
	"github.com/hueristiq/xcrawl3r/pkg/browser"
)

func (crawler *Crawler) pageCrawl(parsedURL *hqgourl.URL) <-chan Result {
	results := make(chan Result)

	go func() {
		defer close(results)

		if crawler.Render {
			// If we're using a proxy send it to the chrome instance
			browser.GlobalContext, browser.GlobalCancel = browser.GetGlobalContext(crawler.Headless, strings.Join(crawler.Proxies, ","))

			// Close the main tab when we end the main() function
			defer browser.GlobalCancel()

			// If renderJavascript, pass the response's body to the renderer and then replace the body for .OnHTML to handle.
			crawler.PageCollector.OnResponse(func(request *colly.Response) {
				html := browser.GetRenderedSource(request.Request.URL.String())

				request.Body = []byte(html)
			})
		}

		crawler.PageCollector.OnRequest(func(request *colly.Request) {
			if match := crawler.URLsNotToRequestRegex.MatchString(request.URL.String()); match {
				request.Abort()

				return
			}

			if match := crawler.FileURLsRegex.MatchString(request.URL.String()); match {
				if err := crawler.FileCollector.Visit(request.URL.String()); err != nil {
					return
				}

				request.Abort()

				return
			}
		})

		crawler.FileCollector.OnError(func(_ *colly.Response, err error) {
			result := Result{
				Type:   ResultError,
				Source: "page",
				Error:  err,
			}

			results <- result
		})

		crawler.PageCollector.OnHTML("[href]", func(e *colly.HTMLElement) {
			relativeURL := e.Attr("href")
			absoluteURL := e.Request.AbsoluteURL(relativeURL)

			parsedAbsoluteURL, err := up.Parse(absoluteURL)
			if err != nil {
				return
			}

			// Ensure we're not visiting a mailto: or similar link
			if !strings.Contains(parsedAbsoluteURL.Scheme, "http") {
				return
			}

			if !crawler.IsInScope(absoluteURL) {
				return
			}

			result := Result{
				Type:   ResultURL,
				Source: "page:href",
				Value:  absoluteURL,
			}

			results <- result

			if err = e.Request.Visit(absoluteURL); err != nil {
				result := Result{
					Type:   ResultError,
					Source: "page:href",
					Error:  err,
				}

				results <- result

				return
			}
		})

		crawler.PageCollector.OnHTML("[src]", func(e *colly.HTMLElement) {
			relativeURL := e.Attr("src")
			absoluteURL := e.Request.AbsoluteURL(relativeURL)

			if !crawler.IsInScope(absoluteURL) {
				return
			}

			result := Result{
				Type:   ResultURL,
				Source: "page:src",
				Value:  absoluteURL,
			}

			results <- result

			if match := crawler.FileURLsRegex.MatchString(absoluteURL); match {
				if err := crawler.FileCollector.Visit(absoluteURL); err != nil {
					result := Result{
						Type:   ResultError,
						Source: "page:src",
						Error:  err,
					}

					results <- result

					return
				}

				return
			}

			if err := e.Request.Visit(absoluteURL); err != nil {
				result := Result{
					Type:   ResultError,
					Source: "page:src",
					Error:  err,
				}

				results <- result

				return
			}
		})

		crawler.FileCollector.OnRequest(func(request *colly.Request) {
			// If the URL is a `.min.js` (Minified JavaScript) try finding `.js`
			if strings.Contains(request.URL.String(), ".min.js") {
				js := strings.ReplaceAll(request.URL.String(), ".min.js", ".js")

				if err := crawler.FileCollector.Visit(js); err != nil {
					return
				}
			}
		})

		crawler.FileCollector.OnResponse(func(response *colly.Response) {
			ext := path.Ext(response.Request.URL.Path)
			body := decode(string(response.Body))
			URLs := crawler.URLsRegex.FindAllString(body, -1)

			for _, fileURL := range URLs {
				// remove beginning and ending quotes
				fileURL = strings.Trim(fileURL, "\"")
				fileURL = strings.Trim(fileURL, "'")

				// remove beginning and ending spaces
				fileURL = strings.Trim(fileURL, " ")

				// ignore, if it's a mime type
				_, _, err := mime.ParseMediaType(fileURL)
				if err == nil {
					continue
				}

				// Get the absolute URL
				fileURL = response.Request.AbsoluteURL(fileURL)

				fileURL = crawler.fixURL(parsedURL, fileURL)

				if !crawler.IsInScope(fileURL) {
					continue
				}

				result := Result{
					Type:   ResultURL,
					Source: "file:" + ext,
					Value:  fileURL,
				}

				results <- result

				if err := crawler.PageCollector.Visit(fileURL); err != nil {
					result := Result{
						Type:   ResultError,
						Source: "file:" + ext,
						Error:  err,
					}

					results <- result

					return
				}
			}
		})

		if err := crawler.PageCollector.Visit(parsedURL.String()); err != nil {
			result := Result{
				Type:   ResultError,
				Source: "page",
				Error:  err,
			}

			results <- result

			return
		}

		crawler.PageCollector.Wait()
		crawler.FileCollector.Wait()
	}()

	return results
}
