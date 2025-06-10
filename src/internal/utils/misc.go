package utils

import (
	"io"
	"net/http"
	"strings"
)

func StartsWith(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func HttpGet(url string) (string, error) {
	resp, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func WrapTextByWords(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = wrapLine(line, maxLen)
	}
	return strings.Join(lines, "\n")
}

func wrapLine(line string, maxLen int) string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return line
	}

	var b strings.Builder
	var lineLen int
	for _, w := range words {
		wLen := len([]rune(w))
		if lineLen == 0 {
			b.WriteString(w)
			lineLen = wLen
		} else if lineLen+1+wLen <= maxLen {
			b.WriteByte(' ')
			b.WriteString(w)
			lineLen += 1 + wLen
		} else {
			b.WriteRune('\n')
			b.WriteString(w)
			lineLen = wLen
		}
	}
	return b.String()
}

func ContainsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
