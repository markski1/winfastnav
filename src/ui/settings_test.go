package ui

import (
	"fmt"
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"winfastnav/internal/documents"
	g "winfastnav/internal/globals"
)

func settingsTestLauncher() *launcher {
	l := &launcher{
		theme: answerTestTheme(), settingsStatus: "Changes are saved automatically.",
		settingsList: widget.List{List: layout.List{Axis: layout.Vertical}},
	}
	l.applyPalette()
	l.indexRoots.SingleLine, l.indexExclusions.SingleLine = true, true
	l.indexRoots.SetText(`C:\Documents; D:\Projects`)
	l.indexExclusions.SetText("node_modules; venv; __pycache__; sdk-manifests; sdk")
	l.savedIndexConfig = documents.IndexConfig{Roots: documents.ParseIndexList(l.indexRoots.Text()), Exclusions: documents.ParseIndexList(l.indexExclusions.Text())}
	return l
}

func settingsTestContext(scale float32) layout.Context {
	router := new(input.Router)
	return layout.Context{
		Ops: new(op.Ops), Source: router.Source(),
		Constraints: layout.Exact(image.Pt(int(560*scale), int(440*scale))),
		Metric:      unit.Metric{PxPerDp: scale, PxPerSp: scale},
	}
}

func TestSettingsPaneNavigation(t *testing.T) {
	l := settingsTestLauncher()
	l.pageList.Position.First = 7
	for _, pane := range []settingsPane{settingsAppearance, settingsIndexing, settingsHiddenApps, settingsGeneral} {
		l.settingsList.Position.First = 10
		l.settingsNav[pane].Click()
		l.settingsPage(settingsTestContext(1))
		if l.settingsPane != pane || l.settingsList.Position.First != 0 {
			t.Fatalf("pane switch failed: pane=%d position=%+v", l.settingsPane, l.settingsList.Position)
		}
		if l.indexRoots.Text() != `C:\Documents; D:\Projects` || l.indexExclusions.Text() != "node_modules; venv; __pycache__; sdk-manifests; sdk" {
			t.Fatal("pane switch discarded folder edits")
		}
		if l.pageList.Position.First != 7 {
			t.Fatal("settings scrolling changed the answer page position")
		}
	}
}

func TestSettingsPanesFit(t *testing.T) {
	previous := g.ExecBlocklist
	g.ExecBlocklist = nil
	defer func() { g.ExecBlocklist = previous }()
	for _, scale := range []float32{1, 1.5, 2} {
		for pane := settingsGeneral; pane < settingsPaneCount; pane++ {
			t.Run(fmt.Sprintf("%s/%.1f", settingsPaneTitles[pane], scale), func(t *testing.T) {
				l := settingsTestLauncher()
				l.settingsPane = pane
				gtx := settingsTestContext(scale)
				dimensions := l.settingsPage(gtx)
				if dimensions.Size != gtx.Constraints.Max {
					t.Fatalf("settings size = %v, want %v", dimensions.Size, gtx.Constraints.Max)
				}
				if l.settingsList.Position.Count != len(l.settingsPaneWidgets()) || l.settingsList.Position.OffsetLast < 0 {
					t.Fatalf("pane controls overflow: %+v", l.settingsList.Position)
				}
			})
		}
	}
}

func TestHiddenAppsScroll(t *testing.T) {
	previous := g.ExecBlocklist
	defer func() { g.ExecBlocklist = previous }()
	g.ExecBlocklist = nil
	for index := 0; index < 20; index++ {
		g.ExecBlocklist = append(g.ExecBlocklist, fmt.Sprintf(`C:\Program Files\Example %d\app.exe`, index))
	}
	l := settingsTestLauncher()
	l.settingsPane = settingsHiddenApps
	l.settingsList.Position.First = 10
	l.settingsPage(settingsTestContext(1))
	if l.settingsList.Position.First == 0 || l.settingsList.Position.Count >= len(l.settingsPaneWidgets()) {
		t.Fatalf("hidden apps did not scroll: %+v", l.settingsList.Position)
	}
	l.settingsNav[settingsGeneral].Click()
	l.settingsPage(settingsTestContext(1))
	if l.settingsList.Position.First != 0 {
		t.Fatal("new pane inherited the hidden apps scroll position")
	}
}
