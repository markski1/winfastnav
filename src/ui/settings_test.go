package ui

import (
	"fmt"
	"testing"
	g "winfastnav/internal/globals"
)

func TestSettingsPaneNavigation(t *testing.T) {
	l := testLauncher(pageSettings)
	l.pageList.Position.First = 7
	for _, pane := range []settingsPane{settingsAppearance, settingsIndexing, settingsHiddenApps, settingsGeneral} {
		l.settingsList.Position.First = 10
		l.settingsNav[pane].Click()
		l.settingsPage(testContext())
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

func TestHiddenAppsScroll(t *testing.T) {
	previous := g.ExecBlocklist
	defer func() { g.ExecBlocklist = previous }()
	g.ExecBlocklist = nil
	for index := 0; index < 20; index++ {
		g.ExecBlocklist = append(g.ExecBlocklist, fmt.Sprintf(`C:\Program Files\Example %d\app.exe`, index))
	}
	l := testLauncher(pageSettings)
	l.settingsPane = settingsHiddenApps
	l.settingsList.Position.First = 10
	l.settingsPage(testContext())
	if l.settingsList.Position.First == 0 || l.settingsList.Position.Count >= len(l.settingsPaneWidgets()) {
		t.Fatalf("hidden apps did not scroll: %+v", l.settingsList.Position)
	}
	l.settingsNav[settingsGeneral].Click()
	l.settingsPage(testContext())
	if l.settingsList.Position.First != 0 {
		t.Fatal("new pane inherited the hidden apps scroll position")
	}
}
