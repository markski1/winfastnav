package ui

import (
	"fmt"
	"testing"

	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/widget"
)

func pagesTestLauncher() *launcher {
	l := &launcher{
		theme: answerTestTheme(), state: uiState{page: pageMenu, selected: -1},
		helpList: widget.List{List: layout.List{Axis: layout.Vertical}},
	}
	l.applyPalette()
	return l
}

func TestHelpPanesFit(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		for pane := helpShortcuts; pane < helpPaneCount; pane++ {
			t.Run(fmt.Sprintf("%s/%.1f", helpPaneTitles[pane], scale), func(t *testing.T) {
				l := pagesTestLauncher()
				l.helpPane = pane
				gtx := settingsTestContext(scale)
				l.helpPage(gtx)
				if l.helpList.Position.Count != len(l.helpPaneWidgets()) || l.helpList.Position.OffsetLast < 0 {
					t.Fatalf("help content overflowed: %+v", l.helpList.Position)
				}
			})
		}
	}
}

func TestHelpPaneNavigation(t *testing.T) {
	l := pagesTestLauncher()
	l.pageList.Position.First = 7
	l.settingsList.Position.First = 3
	for _, pane := range []helpPane{helpSearch, helpTools, helpCommands, helpShortcuts} {
		l.helpList.Position.First = 10
		l.helpNav[pane].Click()
		l.helpPage(settingsTestContext(1))
		if l.helpPane != pane || l.helpList.Position.First != 0 {
			t.Fatalf("pane switch failed: pane=%d position=%+v", l.helpPane, l.helpList.Position)
		}
		if l.pageList.Position.First != 7 || l.settingsList.Position.First != 3 {
			t.Fatal("help changed another page's scroll position")
		}
	}
}

func TestMenuKeyboardNavigation(t *testing.T) {
	l := pagesTestLauncher()
	l.helpPane = helpCommands
	l.helpList.Position.First = 10
	var router input.Router
	gtx := settingsTestContext(1)
	gtx.Source = router.Source()
	frame := func() {
		gtx.Ops.Reset()
		l.update(gtx)
		l.menuPage(gtx)
		router.Frame(gtx.Ops)
	}
	frame()
	gtx.Execute(key.FocusCmd{Tag: &l.help})
	frame()
	router.Queue(key.Event{Name: key.NameReturn, State: key.Press}, key.Event{Name: key.NameReturn, State: key.Release})
	frame()
	if l.snapshot().page != pageHelp || l.helpPane != helpCommands || l.helpList.Position.First != 10 {
		t.Fatalf("Enter did not open Help: state=%+v pane=%d position=%+v", l.snapshot(), l.helpPane, l.helpList.Position)
	}
}
