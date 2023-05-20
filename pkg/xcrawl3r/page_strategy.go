package xcrawl3r

import (
	"mime"
	"path"
	"strings"

	"github.com/gocolly/colly/v2"
	hqurl "github.com/hueristiq/hqgoutils/url"
)

func (crawler *Crawler) pageCrawl(parsedURL *hqurl.URL) (URLsChannel chan URL) {
	URLsChannel = make(chan URL)

	go func() {
		defer close(URLsChannel)

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

		crawler.PageCollector.OnHTML("[href]", func(e *colly.HTMLElement) {
			relativeURL := e.Attr("href")
			absoluteURL := e.Request.AbsoluteURL(relativeURL)

			parsedAbsoluteURL, err := hqurl.Parse(absoluteURL)
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

			URLsChannel <- URL{Source: "page:href", Value: absoluteURL}

			if err = e.Request.Visit(absoluteURL); err != nil {
				return
			}
		})

		crawler.PageCollector.OnHTML("[src]", func(e *colly.HTMLElement) {
			relativeURL := e.Attr("src")
			absoluteURL := e.Request.AbsoluteURL(relativeURL)

			if !crawler.IsInScope(absoluteURL) {
				return
			}

			URLsChannel <- URL{Source: "page:src", Value: absoluteURL}

			if match := crawler.FileURLsRegex.MatchString(absoluteURL); match {
				if err := crawler.FileCollector.Visit(absoluteURL); err != nil {
					return
				}

				return
			}

			if err := e.Request.Visit(absoluteURL); err != nil {
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

			for index := range URLs {
				fileURL := URLs[index]

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

				URLsChannel <- URL{Source: "file:" + ext, Value: fileURL}

				if err := crawler.PageCollector.Visit(fileURL); err != nil {
					return
				}
			}
		})

		if err := crawler.PageCollector.Visit(parsedURL.String()); err != nil {
			return
		}

		crawler.PageCollector.Wait()
		crawler.FileCollector.Wait()
	}()

	return
}
