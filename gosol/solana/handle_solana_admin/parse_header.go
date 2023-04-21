package handle_solana_admin

import (
	"net/http"
	"regexp"
	"strings"
)

var rxNewline = regexp.MustCompile(`[\r\n]+`)

func parseHeader(s string) http.Header {
	h := make(map[string][]string)

	for _, v := range strings.Split(s, "\n") {
		if strings.Index(v, ":") == -1 {
			continue
		}
		v = rxNewline.ReplaceAllString(v, "")

		tmp := strings.Split(v, ":")
		if len(tmp) < 2 {
			continue
		}
		tmp[0] = http.CanonicalHeaderKey(strings.Trim(tmp[0], "\r\n\t "))
		tmp[1] = strings.Trim(tmp[1], "\r\n\t ")
		if len(tmp[0]) == 0 || len(tmp[1]) == 0 {
			continue
		}
		h[tmp[0]] = []string{tmp[1]}
	}
	if len(h) == 0 {
		return nil
	}
	return http.Header(h)
}
