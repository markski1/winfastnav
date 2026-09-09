package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type markdownSpan struct {
	text   string
	url    string
	bold   bool
	italic bool
}

func parseBasicMarkdown(value string) []markdownSpan {
	var spans []markdownSpan
	for len(value) > 0 {
		if strings.HasPrefix(value, "**") {
			if end := strings.Index(value[2:], "**"); end >= 0 {
				spans = append(spans, markdownSpan{text: value[2 : end+2], bold: true})
				value = value[end+4:]
				continue
			}
		}
		if value[0] == '*' {
			if end := strings.Index(value[1:], "*"); end >= 0 {
				spans = append(spans, markdownSpan{text: value[1 : end+1], italic: true})
				value = value[end+2:]
				continue
			}
		}
		if value[0] == '[' {
			if labelEnd := strings.Index(value, "]("); labelEnd > 0 {
				if urlEnd := strings.Index(value[labelEnd+2:], ")"); urlEnd >= 0 {
					url := value[labelEnd+2 : labelEnd+2+urlEnd]
					if isURL(url) {
						spans = append(spans, markdownSpan{text: value[1:labelEnd], url: url})
						value = value[labelEnd+3+urlEnd:]
						continue
					}
				}
			}
		}
		if isURL(value) {
			end := strings.IndexAny(value, " \t\r\n")
			if end < 0 {
				end = len(value)
			}
			url := strings.TrimRight(value[:end], ".,;:!?")
			spans = append(spans, markdownSpan{text: url, url: url})
			value = value[len(url):]
			continue
		}
		length := utf8.RuneLen(rune(value[0]))
		if value[0] >= utf8.RuneSelf {
			_, length = utf8.DecodeRuneInString(value)
		}
		appendMarkdownText(&spans, value[:length])
		value = value[length:]
	}
	return spans
}

func appendMarkdownText(spans *[]markdownSpan, value string) {
	if len(*spans) > 0 {
		last := &(*spans)[len(*spans)-1]
		if last.url == "" && !last.bold && !last.italic {
			last.text += value
			return
		}
	}
	*spans = append(*spans, markdownSpan{text: value})
}

func isURL(value string) bool {
	return strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://")
}

func splitMarkdownText(value string) []string {
	var parts []string
	for len(value) > 0 {
		runeValue, size := utf8.DecodeRuneInString(value)
		if runeValue == '\n' {
			parts = append(parts, "\n")
			value = value[size:]
			continue
		}
		whitespace := unicode.IsSpace(runeValue)
		end := size
		for end < len(value) {
			next, nextSize := utf8.DecodeRuneInString(value[end:])
			if next == '\n' || unicode.IsSpace(next) != whitespace {
				break
			}
			end += nextSize
		}
		parts = append(parts, value[:end])
		value = value[end:]
	}
	return parts
}
