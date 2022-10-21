package hscommon

import (
	"strings"
)

func _prefix_gen(s string, to_len int, with string) string {

	if len(s) >= to_len {
		return ""
	}

	_r := to_len - len(s)
	if len(with) > 1 {
		_r = (_r / len(with)) + 1
	}
	return strings.Repeat(with, _r)[0:_r]
}

func StrPrefix(s string, to_len int, with string) string {
	return _prefix_gen(s, to_len, with) + s
}

func StrPostfix(s string, to_len int, with string) string {
	return s + _prefix_gen(s, to_len, with)
}

func StrPrefixHTML(s string, to_len int, with string) string {
	html_len := len(s) - StrRealLen(s)
	return _prefix_gen(s, to_len+html_len, with) + s
}

func StrPostfixHTML(s string, to_len int, with string) string {
	html_len := len(s) - StrRealLen(s)
	return s + _prefix_gen(s, to_len+html_len, with)
}

func StrRealLen(s string) int {

	if len(s) == 0 {
		return 0
	}

	htmlTagStart := '<'
	htmlTagEnd := '>'

	sr := []rune(s)
	count := 0
	should_count := true
	last_char := sr[0]
	for _, c := range []rune(sr) {
		if c == htmlTagStart {
			should_count = false
		}
		if c == htmlTagEnd {
			should_count = true
			last_char = c
			continue
		}
		if should_count && (c != ' ' || last_char != ' ') {
			count++
		}

		last_char = c
	}
	return count
}

func StripHTML(s string) string {
	ret := ""
	htmlTagStart := '<'
	htmlTagEnd := '>'

	sr := []rune(s)
	should_count := true
	for _, c := range []rune(sr) {
		if c == htmlTagStart {
			should_count = false
		}
		if c == htmlTagEnd {
			should_count = true
			continue
		}
		if should_count {
			ret += string(c)
		}
	}

	return ret
}

func StrMessage(m string, is_ok bool) string {
	if is_ok {
		return "<span style='color: #449944; font-family: monospace'> <b>⬤</b> " + m + "</span>"
	} else {
		return "<span style='color: #dd4444; font-family: monospace'> <b>⮿</b> " + m + "</span>"
	}
}

func StrFirstChars(s string, max_len int) string {
	if max_len == 0 {
		return ""
	}
	if len(s) <= max_len {
		return s
	}
	return s[0:max_len] + "..."
}

func StrLastChars(s string, max_len int) string {
	if max_len == 0 {
		return ""
	}
	if len(s) <= max_len {
		return s
	}
	return "..." + s[len(s)-max_len:]
}

func StrMidChars(s string, max_len int) string {
	if max_len == 0 {
		return ""
	}
	max_len = (max_len + 1) / 2
	if len(s) <= max_len*2 {
		return s
	}
	return s[0:max_len] + "..." + s[len(s)-max_len:]
}
