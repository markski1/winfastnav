package ui

import (
	"image"
	"strings"
	"testing"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/widget/material"
)

func answerTestTheme() *material.Theme {
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	return theme
}

func answerTestContext(width int) layout.Context {
	return layout.Context{Ops: new(op.Ops), Constraints: layout.Constraints{Max: image.Pt(width, 1000)}}
}

func TestAnswerParagraphs(t *testing.T) {
	l := &launcher{theme: answerTestTheme(), answerSpans: parseBasicMarkdown("First paragraph.\n\nSecond paragraph.\nThird line.")}
	lines := l.answerLines(answerTestContext(500))
	if len(lines) != 4 || len(lines[1]) != 0 {
		t.Fatalf("paragraph lines = %#v", lines)
	}
	if len(lines[0]) != 1 || lines[0][0].text != "First paragraph." {
		t.Fatalf("plain text was not shaped together: %#v", lines[0])
	}
}

func TestAnswerWrapping(t *testing.T) {
	for _, value := range []string{
		"Reports say Malaysia began returning people to Myanmar, with about 1,500 expected in the first phase.",
		"Read [a long source title that should wrap across several lines](https://example.com) for more.",
		"https://example.com/" + strings.Repeat("long-path-", 30),
		"A **bold phrase** followed by *italic text* and normal text.",
	} {
		t.Run(value[:min(len(value), 30)], func(t *testing.T) {
			l := &launcher{theme: answerTestTheme(), answerSpans: parseBasicMarkdown(value), answerURLs: []string{"https://example.com"}}
			gtx := answerTestContext(160)
			lines := l.answerLines(gtx)
			if len(lines) < 2 {
				t.Fatal("expected wrapped text")
			}
			var content strings.Builder
			for _, line := range lines {
				width := 0
				for _, atom := range line {
					width += atom.width
					content.WriteString(atom.text)
					if atom.span.url != "" && atom.link != 0 {
						t.Fatalf("wrapped link lost its target: %#v", atom)
					}
				}
				if width > gtx.Constraints.Max.X {
					t.Fatalf("line width = %d, available = %d", width, gtx.Constraints.Max.X)
				}
				content.WriteByte(' ')
			}
			var expected strings.Builder
			for _, span := range l.answerSpans {
				expected.WriteString(span.text)
			}
			if strings.Join(strings.Fields(content.String()), "") != strings.Join(strings.Fields(expected.String()), "") {
				t.Fatalf("wrapped text changed: %q", content.String())
			}
		})
	}
}

func TestAnswerSpaceWidth(t *testing.T) {
	l := &launcher{theme: answerTestTheme()}
	gtx := answerTestContext(500)
	plain := l.answerAtomWidth(gtx, answerAtom{text: "answer"})
	spaced := l.answerAtomWidth(gtx, answerAtom{text: "answer "})
	if spaced <= plain {
		t.Fatalf("trailing space was dropped: %d <= %d", spaced, plain)
	}
}
