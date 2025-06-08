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
	words := strings.Fields(s)
	if maxLen <= 0 || len(words) == 0 {
		return s
	}

	var b strings.Builder
	var lineLen int

	for _, w := range words {
		runes := []rune(w)
		wLen := len(runes)

		if lineLen == 0 {
			// start a new line
			b.WriteString(w)
			lineLen = wLen
		} else if lineLen+1+wLen <= maxLen {
			// append to current line
			b.WriteByte(' ')
			b.WriteString(w)
			lineLen += 1 + wLen
		} else {
			// wrap before this word
			b.WriteRune('\n')
			b.WriteString(w)
			lineLen = wLen
		}
	}

	return b.String()
}
