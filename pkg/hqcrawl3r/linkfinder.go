package hqcrawl3r

import (
	"net/url"
	"regexp"
	"strings"
)

func (crawler *Crawler) FindLinks(source string) (links []string, err error) {
	source, err = url.QueryUnescape(source)
	if err == nil {
		return
	}

	// In case json encoded chars
	replacer := strings.NewReplacer(
		`\u002f`, "/",
		`\u0026`, "&",
	)

	source = replacer.Replace(source)

	lfRegex, err := regexp.Compile(`(?:"|')(((?:[a-zA-Z]{1,10}://|//)[^"'/]{1,}\.[a-zA-Z]{2,}[^"']{0,})|((?:/|\.\./|\./)[^"'><,;| *()(%%$^/\\\[\]][^"'><,;|()]{1,})|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{1,}\.(?:[a-zA-Z]{1,4}|action)(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-/]{1,}/[a-zA-Z0-9_\-/]{3,}(?:[\?|#][^"|']{0,}|))|([a-zA-Z0-9_\-]{1,}\.(?:php|asp|aspx|jsp|json|action|html|js|txt|xml)(?:[\?|#][^"|']{0,}|)))(?:"|')`)
	if err != nil {
		return
	}

	for _, link := range lfRegex.FindAllString(source, -1) {
		links = append(links, link)
	}

	return
}
