package ui

import (
	"testing"

	"gioui.org/io/input"
	"gioui.org/io/key"
)

func TestHelpPaneNavigation(t *testing.T) {
	l := testLauncher(pageMenu)
	l.pageList.Position.First = 7
	l.settingsList.Position.First = 3
	for _, pane := range []helpPane{helpSearch, helpTools, helpCommands, helpShortcuts} {
		l.helpList.Position.First = 10
		l.helpNav[pane].Click()
		l.helpPage(testContext())
		if l.helpPane != pane || l.helpList.Position.First != 0 {
			t.Fatalf("pane switch failed: pane=%d position=%+v", l.helpPane, l.helpList.Position)
		}
		if l.pageList.Position.First != 7 || l.settingsList.Position.First != 3 {
			t.Fatal("help changed another page's scroll position")
		}
	}
}

func TestMenuKeyboardNavigation(t *testing.T) {
	l := testLauncher(pageMenu)
	l.helpPane = helpCommands
	l.helpList.Position.First = 10
	var router input.Router
	gtx := testContext()
	gtx.Source = router.Source()
	frame := func() { testFrame(l, &router, gtx, l.menuPage) }
	frame()
	gtx.Execute(key.FocusCmd{Tag: &l.help})
	frame()
	router.Queue(key.Event{Name: key.NameReturn, State: key.Press}, key.Event{Name: key.NameReturn, State: key.Release})
	frame()
	if l.snapshot().page != pageHelp || l.helpPane != helpCommands || l.helpList.Position.First != 10 {
		t.Fatalf("Enter did not open Help: state=%+v pane=%d position=%+v", l.snapshot(), l.helpPane, l.helpList.Position)
	}
}
