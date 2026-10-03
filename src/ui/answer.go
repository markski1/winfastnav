package ui

import (
	"image"
	"log"
	"strings"
	"unicode/utf8"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/image/math/fixed"
	"winfastnav/internal/utils"
)

const answerTextSize = unit.Sp(15)

func (l *launcher) quickAnswer(gtx layout.Context, message string) layout.Dimensions {
	for l.answerCopy.Clicked(gtx) {
		l.copyAnswer(gtx)
	}
	for l.answerRetry.Clicked(gtx) {
		state := l.snapshot()
		if !state.loading && state.answerPrompt != "" {
			l.askAssistant(state.answerPrompt)
		}
	}
	state := l.snapshot()
	if state.answer {
		message = state.message
	}
	if l.answerMessage != message {
		l.answerMessage = message
		l.answerSpans = parseBasicMarkdown(strings.ReplaceAll(message, "\r\n", "\n"))
		l.answerURLs = l.answerURLs[:0]
		l.answerLinks = [maxAnswerLinks]widget.Clickable{}
		for _, span := range l.answerSpans {
			if span.url != "" && len(l.answerURLs) < maxAnswerLinks {
				l.answerURLs = append(l.answerURLs, span.url)
			}
		}
		l.pageList.Position = layout.Position{}
	}
	for index, url := range l.answerURLs {
		for l.answerLinks[index].Clicked(gtx) {
			go func(url string) {
				if err := utils.OpenURI(url); err != nil {
					log.Printf("failed to open answer link: %v", err)
				}
			}(url)
		}
	}

	return layout.Inset{Top: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							style := material.Label(l.theme, unit.Sp(11), "Quick answer")
							style.Font.Weight = font.Medium
							style.Color = l.palette.accent
							return style.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if state.loading {
								size := gtx.Dp(16)
								gtx.Constraints = layout.Exact(image.Pt(size, size))
								loader := material.Loader(l.theme)
								loader.Color = l.palette.accent
								return loader.Layout(gtx)
							}
							label := "Copy answer"
							if state.answerCopied {
								label = "Copied"
							}
							return l.button(gtx, &l.answerCopy, label)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if state.loading {
								return layout.Dimensions{}
							}
							return l.button(gtx, &l.answerRetry, "Retry")
						}),
					)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if state.answerPrompt == "" {
					return layout.Dimensions{}
				}
				return layout.Inset{Bottom: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					style := material.Label(l.theme, unit.Sp(12), state.answerPrompt)
					style.Color = l.palette.secondary
					style.MaxLines, style.Truncator = 2, "…"
					return style.Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				list := material.List(l.theme, &l.pageList)
				content := gtx
				content.Constraints.Max.X = max(0, content.Constraints.Max.X-gtx.Dp(list.Width()))
				lines := l.answerLines(content)
				return list.Layout(gtx, len(lines), func(gtx layout.Context, index int) layout.Dimensions {
					line := lines[index]
					if len(line) == 0 {
						return layout.Spacer{Height: unit.Dp(10)}.Layout(gtx)
					}
					return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						children := make([]layout.FlexChild, 0, len(line))
						for _, atom := range line {
							children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return l.answerAtom(gtx, atom)
							}))
						}
						return layout.Flex{Alignment: layout.Baseline}.Layout(gtx, children...)
					})
				})
			}),
		)
	})
}

func answerPlainText(message string) string {
	var value strings.Builder
	for _, span := range parseBasicMarkdown(strings.ReplaceAll(message, "\r\n", "\n")) {
		value.WriteString(span.text)
		if span.url != "" && span.text != span.url {
			value.WriteString(" (" + span.url + ")")
		}
	}
	return value.String()
}

func (l *launcher) copyAnswer(gtx layout.Context) {
	state := l.snapshot()
	if !state.answer || state.loading || state.message == "" {
		return
	}
	l.copyText(gtx, answerPlainText(state.message))
	l.updateState(func(state *uiState) { state.answerCopied = true })
}

func (l *launcher) answerLines(gtx layout.Context) [][]answerAtom {
	var lines [][]answerAtom
	var line []answerAtom
	width, linkIndex := 0, 0
	for _, span := range l.answerSpans {
		link := -1
		if span.url != "" {
			if linkIndex < len(l.answerURLs) {
				link = linkIndex
			}
			linkIndex++
		}
		for _, part := range splitMarkdownText(span.text) {
			if part == "\n" {
				lines = append(lines, line)
				line, width = nil, 0
				continue
			}
			l.addAnswerAtom(gtx, &lines, &line, &width, answerAtom{text: part, span: span, link: link})
		}
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}
	return lines
}

func (l *launcher) addAnswerAtom(gtx layout.Context, lines *[][]answerAtom, line *[]answerAtom, width *int, atom answerAtom) {
	if len(*line) == 0 && strings.TrimSpace(atom.text) == "" {
		return
	}
	atom.width = l.answerAtomWidth(gtx, atom)
	merged := atom
	merge := false
	previousWidth := 0
	if len(*line) > 0 {
		previous := (*line)[len(*line)-1]
		if previous.span == atom.span && previous.link == atom.link {
			merge = true
			previousWidth = previous.width
			merged.text = previous.text + atom.text
			merged.width = l.answerAtomWidth(gtx, merged)
		}
	}
	if len(*line) > 0 && *width-previousWidth+merged.width > gtx.Constraints.Max.X {
		*lines = append(*lines, *line)
		*line, *width = nil, 0
		l.addAnswerAtom(gtx, lines, line, width, atom)
		return
	}
	if atom.width > gtx.Constraints.Max.X && utf8.RuneCountInString(atom.text) > 1 {
		for _, value := range atom.text {
			part := atom
			part.text = string(value)
			l.addAnswerAtom(gtx, lines, line, width, part)
		}
		return
	}
	if merge {
		(*line)[len(*line)-1] = merged
		*width += merged.width - previousWidth
	} else {
		*line = append(*line, atom)
		*width += atom.width
	}
}

func (l *launcher) answerAtomStyle(atom answerAtom) material.LabelStyle {
	style := material.Label(l.theme, answerTextSize, atom.text)
	style.Color = l.palette.text
	style.MaxLines = 1
	if atom.span.bold {
		style.Font.Weight = font.Bold
	}
	if atom.span.italic {
		style.Font.Style = font.Italic
	}
	if atom.span.url != "" {
		style.Color = l.palette.accent
	}
	return style
}

func (l *launcher) answerAtomWidth(gtx layout.Context, atom answerAtom) int {
	style := l.answerAtomStyle(atom)
	l.theme.Shaper.LayoutString(text.Parameters{
		Font: style.Font, PxPerEm: fixed.I(gtx.Sp(style.TextSize)),
		MaxWidth: 1 << 20, Locale: gtx.Locale, DisableSpaceTrim: true,
	}, atom.text)
	width := 0
	for glyph, ok := l.theme.Shaper.NextGlyph(); ok; glyph, ok = l.theme.Shaper.NextGlyph() {
		width = max(width, (glyph.X + glyph.Advance).Ceil())
	}
	return width
}

func (l *launcher) answerAtom(gtx layout.Context, atom answerAtom) layout.Dimensions {
	gtx.Constraints.Min.X = atom.width
	gtx.Constraints.Max.X = atom.width
	style := l.answerAtomStyle(atom)
	if atom.link >= 0 {
		return l.answerLinks[atom.link].Layout(gtx, style.Layout)
	}
	return style.Layout(gtx)
}
