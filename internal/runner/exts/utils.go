package exts

import (
	"net/url"
	"strings"
)

func decodeChars(s string) string {
	source, err := url.QueryUnescape(s)
	if err == nil {
		s = source
	}

	// In case json encoded chars
	replacer := strings.NewReplacer(
		`\u002f`, "/",
		`\u0026`, "&",
	)
	s = replacer.Replace(strings.ToLower(s))
	return s
}
