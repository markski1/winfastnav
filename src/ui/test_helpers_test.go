package ui

import (
	"image"

	"gioui.org/font/gofont"
	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"winfastnav/internal/documents"
)

func answerTestTheme() *material.Theme {
	theme := material.NewTheme()
	theme.TextSize = unit.Sp(12.35)
	theme.Shaper = text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
	return theme
}

func testLauncher(page page) *launcher {
	l := &launcher{theme: answerTestTheme(), state: uiState{page: page, selected: -1}}
	l.applyPalette()
	l.list.Axis, l.pageList.Axis = layout.Vertical, layout.Vertical
	l.settingsList.Axis, l.helpList.Axis = layout.Vertical, layout.Vertical
	l.indexRoots.SingleLine, l.indexExclusions.SingleLine = true, true
	l.indexRoots.SetText(`C:\Documents; D:\Projects`)
	l.indexExclusions.SetText("node_modules; venv; __pycache__; sdk-manifests; sdk")
	l.savedIndexConfig = documents.IndexConfig{Roots: documents.ParseIndexList(l.indexRoots.Text()), Exclusions: documents.ParseIndexList(l.indexExclusions.Text())}
	return l
}

func testContext() layout.Context {
	router := new(input.Router)
	return layout.Context{
		Ops: new(op.Ops), Source: router.Source(),
		Constraints: layout.Exact(image.Pt(560, 440)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
}

func testFrame(l *launcher, router *input.Router, gtx layout.Context, content layout.Widget) {
	gtx.Ops.Reset()
	l.update(gtx)
	content(gtx)
	router.Frame(gtx.Ops)
}
