package ui

import (
	"image"
	"slices"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	g "winfastnav/internal/globals"
)

func TestResultActionAvailability(t *testing.T) {
	for _, tc := range []struct {
		name string
		item g.Resource
		want []resultAction
	}{
		{"app", g.Resource{Filepath: `C:\Apps\app.exe`}, []resultAction{resultOpen, resultReveal, resultCopy, resultAdmin, resultHide}},
		{"shortcut", g.Resource{Filepath: `C:\Apps\app.lnk`}, []resultAction{resultOpen, resultReveal, resultCopy, resultHide}},
		{"store app", g.Resource{Filepath: `shell:AppsFolder\Example!App`}, []resultAction{resultOpen, resultCopy, resultHide}},
		{"document", g.Resource{Filepath: `C:\Docs\report.pdf`, Document: true}, []resultAction{resultOpen, resultReveal, resultCopy}},
		{"computed", g.Resource{Name: "42", Computed: true}, []resultAction{resultCopy}},
		{"assistant", g.Resource{Assistant: "question"}, []resultAction{resultOpen}},
		{"web", g.Resource{WebSearch: "search"}, []resultAction{resultOpen}},
		{"command", g.Resource{Command: &g.SystemCommand{Action: "lock"}}, []resultAction{resultOpen}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []resultAction
			for _, choice := range actionsForResult(tc.item) {
				got = append(got, choice.action)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("actions = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResultCopyShortcutUsesSameRulesAsMenu(t *testing.T) {
	for _, menu := range []bool{false, true} {
		for _, item := range []g.Resource{
			{Name: "42", Computed: true},
			{Filepath: `C:\Docs\report.pdf`, Document: true},
			{Filepath: `shell:AppsFolder\Example!App`},
			{Assistant: "question", Filepath: "should not copy"},
		} {
			l := testLauncher(pageLauncher)
			l.items, l.state.selected = []g.Resource{item}, 0
			if menu {
				l.openResultActions(0)
			}
			var router input.Router
			gtx := testContext()
			gtx.Source = router.Source()
			l.key(gtx, key.Event{Name: "C", Modifiers: key.ModCtrl})
			_, data, copied := router.WriteClipboard()
			want := item.Filepath
			if item.Computed {
				want = item.Name
			}
			if item.Assistant != "" {
				want = ""
			}
			if string(data) != want || copied != (want != "") {
				t.Fatalf("menu=%v item=%+v: copied=%v value=%q", menu, item, copied, data)
			}
		}
	}
}

func TestResultMenuKeepsClickedTarget(t *testing.T) {
	l := testLauncher(pageMenu)
	l.state.page = pageLauncher
	l.items = []g.Resource{{Name: "42", Computed: true}}
	l.openResultActions(0)
	l.items[0] = g.Resource{Name: "99", Computed: true}
	var router input.Router
	gtx := testContext()
	gtx.Source = router.Source()
	l.refreshPending.Store(true)
	l.update(gtx)
	if !l.snapshot().actionMenu || !l.refreshPending.Load() {
		t.Fatal("background refresh interrupted the menu")
	}
	l.key(gtx, key.Event{Name: "C", Modifiers: key.ModCtrl})
	_, value, copied := router.WriteClipboard()
	if !copied || string(value) != "42" || l.snapshot().actionMenu {
		t.Fatalf("menu copied %q instead of clicked target", value)
	}
}

func TestResultMenuPointerAndKeyboard(t *testing.T) {
	for _, secondary := range []bool{true, false} {
		l := testLauncher(pageMenu)
		l.state = uiState{page: pageLauncher, resultCount: 1, selected: 0}
		l.items = []g.Resource{{Name: "42", Computed: true}}
		l.list.Axis = layout.Vertical
		var router input.Router
		gtx := testContext()
		gtx.Source = router.Source()
		frame := func() { testFrame(l, &router, gtx, l.layout) }
		frame()
		buttons, position := pointer.ButtonSecondary, f32.Pt(100, 80)
		if !secondary {
			buttons, position = pointer.ButtonPrimary, f32.Pt(518, 82)
		}
		router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: buttons, Position: position}, pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: position})
		frame()
		frame()
		if !l.snapshot().actionMenu {
			t.Fatalf("secondary=%v did not open menu", secondary)
		}
		if _, _, copied := router.WriteClipboard(); copied {
			t.Fatal("opening menu also activated the result")
		}
		if secondary && l.pointerPosition != image.Pt(100, 80) {
			t.Fatalf("menu anchor = %v", l.pointerPosition)
		}
		router.Queue(key.Event{Name: key.NameReturn, State: key.Press}, key.Event{Name: key.NameReturn, State: key.Release})
		frame()
		frame()
		_, value, copied := router.WriteClipboard()
		if !copied || string(value) != "42" || l.snapshot().actionMenu {
			t.Fatalf("Enter did not activate focused menu action: copied=%v value=%q", copied, value)
		}
	}
}

func TestPopupStaysWithinWindow(t *testing.T) {
	for _, point := range []image.Point{image.Pt(-20, -20), image.Pt(20, 30), image.Pt(550, 430)} {
		position := popupPosition(point, image.Pt(300, 230), image.Pt(560, 440), 10)
		if position.X < 10 || position.Y < 10 || position.X+300 > 550 || position.Y+230 > 430 {
			t.Fatalf("popup outside window: %v", position)
		}
	}
}

func TestResultMenuKeyboardNavigationAndDismissal(t *testing.T) {
	l := testLauncher(pageMenu)
	l.state = uiState{page: pageLauncher, selected: 0}
	l.items = []g.Resource{{Name: "Report", Filepath: `C:\Docs\report.pdf`, Document: true}}
	var router input.Router
	gtx := testContext()
	gtx.Source = router.Source()
	frame := func() {
		gtx.Ops.Reset()
		l.update(gtx)
		if l.snapshot().actionMenu {
			l.resultActionMenu(gtx)
		}
		router.Frame(gtx.Ops)
	}
	l.key(gtx, key.Event{Name: key.NameF10, Modifiers: key.ModShift})
	frame()
	for range 2 {
		router.Queue(key.Event{Name: key.NameDownArrow, State: key.Press})
		frame()
	}
	router.Queue(key.Event{Name: key.NameReturn, State: key.Press}, key.Event{Name: key.NameReturn, State: key.Release})
	frame()
	frame()
	_, value, copied := router.WriteClipboard()
	if !copied || string(value) != `C:\Docs\report.pdf` {
		t.Fatalf("keyboard menu copied %q", value)
	}
	l.openResultActions(0)
	frame()
	router.Queue(key.Event{Name: key.NameEscape, State: key.Press})
	frame()
	if l.snapshot().actionMenu {
		t.Fatal("Escape did not dismiss the menu")
	}
	l.openResultActions(0)
	frame()
	point := f32.Pt(540, 410)
	router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: point}, pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: point})
	frame()
	frame()
	if l.snapshot().actionMenu {
		t.Fatal("outside click did not dismiss the menu")
	}
}
