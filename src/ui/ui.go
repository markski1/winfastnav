package ui

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gioui.org/app"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"winfastnav/internal/apps"
	"winfastnav/internal/core"
	"winfastnav/internal/documents"
	g "winfastnav/internal/globals"
	appicons "winfastnav/internal/icons"
	"winfastnav/internal/recent"
	appsettings "winfastnav/internal/settings"
	"winfastnav/internal/systemactions"
	"winfastnav/internal/utils"
	"winfastnav/internal/windowcontrol"
)

const (
	maxResults     = 30
	maxAnswerLinks = 12
	searchDebounce = 50 * time.Millisecond
)

type launcher struct {
	windowControl                                 *windowcontrol.Controller
	window                                        app.Window
	ops                                           op.Ops
	theme                                         *material.Theme
	icons                                         *appicons.Cache
	editor, settings                              widget.Editor
	indexRoots, indexExclusions                   widget.Editor
	list                                          widget.List
	pageList                                      widget.List
	settingsList                                  widget.List
	helpList                                      widget.List
	helpPane                                      helpPane
	helpNav                                       [helpPaneCount]widget.Clickable
	settingsPane                                  settingsPane
	settingsNav                                   [settingsPaneCount]widget.Clickable
	themeChoices, densityChoices                  [2]widget.Clickable
	textSizeChoices                               [3]widget.Clickable
	textSizeIndex                                 int
	startupSwitch                                 widget.Bool
	startupSet                                    func(bool) error
	savedIndexConfig                              documents.IndexConfig
	results                                       [maxResults + 2]widget.Clickable
	resultMore                                    [maxResults + 2]widget.Clickable
	resultContext                                 [maxResults + 2]int
	rootPointer                                   int
	pointerPosition                               image.Point
	actionButtons                                 [5]widget.Clickable
	actionDismiss                                 widget.Clickable
	actionFocus                                   bool
	menu, back, help, settingsButton, about, quit widget.Clickable
	confirm, cancel                               widget.Clickable
	clearSearch                                   widget.Clickable
	reindex                                       widget.Clickable
	mu                                            sync.RWMutex
	items                                         []g.Resource
	startupEnabled                                bool
	settingsStatus                                string
	unblockButtons                                map[string]*widget.Clickable
	centered                                      bool
	focused                                       atomic.Bool
	windowInitialized                             bool
	lightTheme                                    bool
	compact                                       bool
	palette                                       uiPalette
	windowMu                                      sync.Mutex
	windowReady                                   atomic.Bool
	windowGeneration                              atomic.Uint64
	refreshPending                                atomic.Bool
	answerGeneration                              atomic.Uint64
	answerMessage                                 string
	answerSpans                                   []markdownSpan
	answerURLs                                    []string
	answerLinks                                   [maxAnswerLinks]widget.Clickable
	answerCopy, answerRetry                       widget.Clickable
	answerFetch                                   func(string) string
	pendingAction                                 g.Resource
	searchMu                                      sync.Mutex
	searchGeneration                              uint64
	searchCancel                                  context.CancelFunc
	searchTimer                                   *time.Timer
	pendingSearch                                 *searchResult
	stateMu                                       sync.RWMutex
	state                                         uiState
}

var active *launcher

type resultRow struct {
	title         string
	detail        string
	iconPath      string
	kind          string
	resourceIndex int
	section       bool
}

type searchResult struct {
	generation uint64
	query      string
	items      []g.Resource
	message    string
}

type page uint8

const (
	pageLauncher page = iota
	pageMenu
	pageHelp
	pageSettings
	pageAbout
	pageConfirmation
)

type uiState struct {
	visible      bool
	query        string
	message      string
	loading      bool
	answer       bool
	page         page
	resultCount  int
	selected     int
	focusSearch  bool
	answerPrompt string
	answerCopied bool
	actionMenu   bool
	actionTarget g.Resource
}

type answerAtom struct {
	text  string
	span  markdownSpan
	link  int
	width int
}

type uiPalette struct {
	window      color.NRGBA
	surface     color.NRGBA
	surfaceEdge color.NRGBA
	input       color.NRGBA
	row         color.NRGBA
	selected    color.NRGBA
	hover       color.NRGBA
	text        color.NRGBA
	secondary   color.NRGBA
	muted       color.NRGBA
	accent      color.NRGBA
	button      color.NRGBA
	buttonText  color.NRGBA
	icon        color.NRGBA
}

func darkPalette() uiPalette {
	return uiPalette{
		window:      color.NRGBA{R: 0x0d, G: 0x0f, B: 0x14, A: 0xff},
		surface:     color.NRGBA{R: 0x1b, G: 0x1e, B: 0x26, A: 0xff},
		surfaceEdge: color.NRGBA{R: 0x3a, G: 0x40, B: 0x4e, A: 0xff},
		input:       color.NRGBA{R: 0x28, G: 0x2d, B: 0x38, A: 0xff},
		row:         color.NRGBA{R: 0x21, G: 0x24, B: 0x2c, A: 0xff},
		selected:    color.NRGBA{R: 0x3d, G: 0x68, B: 0x96, A: 0xff},
		hover:       color.NRGBA{R: 0x2d, G: 0x38, B: 0x4a, A: 0xff},
		text:        color.NRGBA{R: 0xf5, G: 0xf7, B: 0xfa, A: 0xff},
		secondary:   color.NRGBA{R: 0xb6, G: 0xbd, B: 0xc9, A: 0xff},
		muted:       color.NRGBA{R: 0x8d, G: 0x96, B: 0xa5, A: 0xff},
		accent:      color.NRGBA{R: 0x78, G: 0xb7, B: 0xff, A: 0xff},
		button:      color.NRGBA{R: 0x36, G: 0x4f, B: 0x70, A: 0xff},
		buttonText:  color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		icon:        color.NRGBA{R: 0x4d, G: 0x61, B: 0x7d, A: 0xff},
	}
}

func lightPalette() uiPalette {
	return uiPalette{
		window:      color.NRGBA{R: 0xe9, G: 0xed, B: 0xf3, A: 0xff},
		surface:     color.NRGBA{R: 0xfc, G: 0xfd, B: 0xff, A: 0xff},
		surfaceEdge: color.NRGBA{R: 0xc8, G: 0xd0, B: 0xdc, A: 0xff},
		input:       color.NRGBA{R: 0xf0, G: 0xf3, B: 0xf8, A: 0xff},
		row:         color.NRGBA{R: 0xf6, G: 0xf8, B: 0xfb, A: 0xff},
		selected:    color.NRGBA{R: 0xd4, G: 0xe8, B: 0xff, A: 0xff},
		hover:       color.NRGBA{R: 0xe8, G: 0xf1, B: 0xfc, A: 0xff},
		text:        color.NRGBA{R: 0x1c, G: 0x24, B: 0x30, A: 0xff},
		secondary:   color.NRGBA{R: 0x5b, G: 0x66, B: 0x75, A: 0xff},
		muted:       color.NRGBA{R: 0x7b, G: 0x86, B: 0x96, A: 0xff},
		accent:      color.NRGBA{R: 0x2b, G: 0x6f, B: 0xb8, A: 0xff},
		button:      color.NRGBA{R: 0x3d, G: 0x73, B: 0xae, A: 0xff},
		buttonText:  color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		icon:        color.NRGBA{R: 0x6b, G: 0x91, B: 0xb8, A: 0xff},
	}
}

func SetupUI() {
	theme := material.NewTheme()
	theme.TextSize = unit.Sp(12.35)
	themeSetting := appsettings.GetSetting("theme")
	densitySetting := appsettings.GetSetting("density")
	textSizeSetting := appsettings.GetSetting("textsize")
	active = &launcher{
		theme:          theme,
		icons:          appicons.NewCache(),
		list:           widget.List{List: layout.List{Axis: layout.Vertical}},
		pageList:       widget.List{List: layout.List{Axis: layout.Vertical}},
		settingsList:   widget.List{List: layout.List{Axis: layout.Vertical}},
		helpList:       widget.List{List: layout.List{Axis: layout.Vertical}},
		lightTheme:     strings.EqualFold(themeSetting, "light"),
		compact:        strings.EqualFold(densitySetting, "compact"),
		textSizeIndex:  textSizeIndexForSetting(textSizeSetting),
		unblockButtons: make(map[string]*widget.Clickable),
		state: uiState{
			visible:     true,
			page:        pageLauncher,
			selected:    -1,
			focusSearch: true,
		},
	}
	active.icons.SetChangedHandler(active.invalidate)
	active.applyPalette()
	active.editor.SingleLine, active.editor.Submit = true, true
	active.indexRoots.SingleLine, active.indexExclusions.SingleLine = true, true
	active.window.Option(app.Title(g.AppName), app.Size(unit.Dp(580), unit.Dp(460)), app.MinSize(unit.Dp(580), unit.Dp(460)), app.MaxSize(unit.Dp(580), unit.Dp(460)), app.Decorated(false), app.TopMost(true))
	active.windowControl = windowcontrol.New(g.AppName)
	active.showRecent()
}

func (l *launcher) applyPalette() {
	if l.lightTheme {
		l.palette = lightPalette()
	} else {
		l.palette = darkPalette()
	}
	l.theme.Palette = material.Palette{
		Fg:         l.palette.text,
		Bg:         l.palette.surface,
		ContrastBg: l.palette.accent,
		ContrastFg: l.palette.buttonText,
	}
}

func Run() {
	if active == nil {
		return
	}
	if err := active.run(); err != nil {
		log.Printf("Gio window closed: %v", err)
	}
}

func ShowWindow() {
	if active == nil {
		return
	}
	generation := active.windowGeneration.Add(1)
	if active.snapshot().page == pageSettings {
		active.commitIndexSettings(false)
	}
	active.prepareShow()
	active.cancelSearch()
	if !active.snapshot().answer {
		active.showRecent()
	}
	active.showAndFocus(generation)
}

func (l *launcher) showAndFocus(generation uint64) {
	if err := l.setWindowVisible(generation, true); err == nil {
		return
	}
	time.AfterFunc(100*time.Millisecond, func() {
		err := l.setWindowVisible(generation, true)
		if err == nil {
			return
		}
		if generation == l.windowGeneration.Load() && l.snapshot().visible {
			log.Printf("failed to show launcher: %v", err)
		}
	})
}

func (l *launcher) setWindowVisible(generation uint64, visible bool) error {
	l.windowMu.Lock()
	defer l.windowMu.Unlock()
	if generation != l.windowGeneration.Load() || l.snapshot().visible != visible {
		return nil
	}
	if visible {
		return l.windowControl.ShowAndFocus()
	}
	return l.windowControl.Hide()
}

func ToggleWindow() {
	if active == nil {
		return
	}
	if active.snapshot().visible {
		HideWindow()
		return
	}
	ShowWindow()
}

func HideWindow() {
	if active == nil {
		return
	}
	generation := active.windowGeneration.Add(1)
	active.cancelSearch()
	active.focused.Store(false)
	active.prepareHide()
	go active.hideWindow(generation)
}

func (l *launcher) hideWindow(generation uint64) {
	if err := l.setWindowVisible(generation, false); err != nil {
		log.Printf("failed to hide launcher: %v", err)
	}
}

func RefreshResults() {
	if active == nil {
		return
	}
	active.refreshPending.Store(true)
	active.invalidate()
}

func RefreshResources() {
	if active == nil {
		return
	}
	active.icons.Clear()
	RefreshResults()
}

func (l *launcher) invalidate() {
	if l.windowReady.Load() {
		l.window.Invalidate()
	}
}

func (l *launcher) snapshot() uiState {
	l.stateMu.RLock()
	defer l.stateMu.RUnlock()
	return l.state
}

func (l *launcher) updateState(update func(*uiState)) uiState {
	l.stateMu.Lock()
	update(&l.state)
	state := l.state
	l.stateMu.Unlock()
	l.invalidate()
	return state
}

func Quit() {
	if active != nil {
		if active.snapshot().page == pageSettings {
			active.commitIndexSettings(false)
		}
		active.cancelSearch()
	}
	if err := appsettings.Flush(); err != nil {
		log.Printf("failed to save settings on exit: %v", err)
	}
	os.Exit(0)
}

func (l *launcher) run() error {
	for {
		switch e := l.window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.ConfigEvent:
			if !l.centered {
				l.window.Perform(system.ActionCenter)
				l.centered = true
			}
			if e.Config.Focused {
				l.focused.Store(true)
			} else if l.focused.Swap(false) && l.snapshot().visible {
				HideWindow()
			}
		case app.ViewEvent:
			l.windowControl.BindView(e)
			if !l.windowInitialized {
				l.windowInitialized = true
				l.focused.Store(false)
				go l.initializeWindow(l.windowGeneration.Load())
			}
		case app.FrameEvent:
			gtx := app.NewContext(&l.ops, e)
			l.windowReady.Store(true)
			l.update(gtx)
			l.layout(gtx)
			e.Frame(&l.ops)
		}
	}
}

func (l *launcher) initializeWindow(generation uint64) {
	if err := l.windowControl.HideFromTaskbar(); err != nil {
		log.Printf("failed to hide launcher from taskbar: %v", err)
	}
	if err := l.setWindowVisible(generation, true); err != nil {
		log.Printf("failed to show launcher on startup: %v", err)
	}
}

func (l *launcher) update(gtx layout.Context) {
	if l.refreshPending.Swap(false) {
		state := l.snapshot()
		if state.actionMenu {
			l.refreshPending.Store(true)
		} else if state.page == pageLauncher && !state.answer {
			l.query(state.query)
		}
	}
	l.applyPendingSearch()
	state := l.snapshot()
	if l.editor.Text() != state.query {
		l.editor.SetText(state.query)
	}
	keyFilters := []event.Filter{key.Filter{Name: key.NameEscape}}
	if state.page == pageLauncher && !state.actionMenu {
		keyFilters = append(keyFilters,
			key.Filter{Name: key.NameUpArrow},
			key.Filter{Name: key.NameDownArrow},
			key.Filter{Name: key.NameReturn},
			key.Filter{Name: key.NameEnter},
			key.Filter{Name: key.NameDeleteForward},
			key.Filter{Name: key.NameHome},
			key.Filter{Name: key.NameEnd},
			key.Filter{Name: key.NamePageUp},
			key.Filter{Name: key.NamePageDown},
			key.Filter{Name: key.NameReturn, Required: key.ModAlt},
			key.Filter{Name: key.NameEnter, Required: key.ModAlt},
			key.Filter{Name: key.NameReturn, Required: key.ModCtrl},
			key.Filter{Name: key.NameEnter, Required: key.ModCtrl},
			key.Filter{Name: key.NameReturn, Required: key.ModShift},
			key.Filter{Name: key.NameEnter, Required: key.ModShift},
			key.Filter{Name: "C", Required: key.ModCtrl},
			key.Filter{Name: "C", Required: key.ModCtrl | key.ModShift},
			key.Filter{Name: key.NameF10, Required: key.ModShift},
		)
	} else if state.actionMenu {
		keyFilters = append(keyFilters,
			key.Filter{Name: key.NameUpArrow},
			key.Filter{Name: key.NameDownArrow},
			key.Filter{Name: "C", Required: key.ModCtrl},
			key.Filter{Name: key.NameReturn, Required: key.ModCtrl},
			key.Filter{Name: key.NameEnter, Required: key.ModCtrl},
			key.Filter{Name: key.NameReturn, Required: key.ModShift},
			key.Filter{Name: key.NameEnter, Required: key.ModShift},
			key.Filter{Name: key.NameDeleteForward},
		)
	} else if state.page == pageConfirmation {
		keyFilters = append(keyFilters, key.Filter{Name: key.NameReturn}, key.Filter{Name: key.NameEnter})
	}
	for {
		e, ok := gtx.Source.Event(keyFilters...)
		if !ok {
			break
		}
		if k, ok := e.(key.Event); ok && k.State == key.Press {
			l.key(gtx, k)
		}
	}
	for {
		e, ok := gtx.Event(pointer.Filter{Target: &l.rootPointer, Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := e.(pointer.Event); ok {
			l.pointerPosition = image.Pt(int(e.Position.X), int(e.Position.Y))
		}
	}
	if l.snapshot().page != pageLauncher {
		return
	}
	for {
		e, ok := l.editor.Update(gtx)
		if !ok {
			break
		}
		switch e.(type) {
		case widget.ChangeEvent:
			l.query(l.editor.Text())
		case widget.SubmitEvent:
			l.submit(gtx, l.editor.Text())
		}
	}
}

func (l *launcher) key(gtx layout.Context, event key.Event) {
	s := l.snapshot()
	if s.actionMenu {
		if event.Name == key.NameUpArrow || event.Name == key.NameDownArrow {
			choices := actionsForResult(s.actionTarget)
			index := 0
			for i := range choices {
				if gtx.Focused(&l.actionButtons[i]) {
					index = i
					break
				}
			}
			if event.Name == key.NameUpArrow {
				index--
			} else {
				index++
			}
			index = (index + len(choices)) % len(choices)
			gtx.Execute(key.FocusCmd{Tag: &l.actionButtons[index]})
		}
		if event.Name == key.NameEscape {
			l.closeResultActions()
		}
		if action, ok := resultShortcut(event); ok {
			l.executeResultAction(gtx, s.actionTarget, action)
		}
		return
	}
	if s.page == pageConfirmation {
		switch event.Name {
		case key.NameEscape:
			l.launcher()
		case key.NameReturn, key.NameEnter:
			l.executeSystemAction(l.pendingAction)
		}
		return
	}
	if s.page != pageLauncher {
		if event.Name == key.NameEscape {
			l.backPage()
		}
		return
	}
	if event.Name == key.NameF10 && event.Modifiers.Contain(key.ModShift) {
		l.pointerPosition = image.Pt(gtx.Dp(24), gtx.Dp(70))
		l.openResultActions(s.selected)
		return
	}
	if action, ok := resultShortcut(event); ok {
		if action == resultCopy && s.answer && !s.loading {
			l.copyAnswer(gtx)
		} else if item, ok := l.itemAt(s.selected); ok {
			l.executeResultAction(gtx, item, action)
		}
		return
	}

	switch event.Name {
	case key.NameEscape:
		if s.page == pageLauncher {
			HideWindow()
		} else {
			l.launcher()
		}
	case key.NameUpArrow:
		l.selectResult(s.selected - 1)
	case key.NameDownArrow:
		l.selectResult(s.selected + 1)
	case key.NameReturn, key.NameEnter:
		if s.selected >= 0 {
			l.open(gtx, s.selected)
		} else {
			l.submit(gtx, l.editor.Text())
		}
	case key.NameHome:
		l.selectResult(0)
	case key.NameEnd:
		l.selectResult(s.resultCount - 1)
	case key.NamePageUp:
		l.selectResult(s.selected - max(l.list.Position.Count-2, 1))
	case key.NamePageDown:
		l.selectResult(s.selected + max(l.list.Position.Count-2, 1))
	}
}

func (l *launcher) query(query string) {
	l.answerGeneration.Add(1)
	l.updateState(func(state *uiState) {
		state.query = query
		state.selected = -1
		state.answer = false
		state.answerPrompt = ""
		state.answerCopied = false
		state.actionMenu = false
	})

	trimmed := strings.TrimSpace(query)
	var shortcutMessage string
	if strings.HasPrefix(trimmed, ">") {
		value := strings.TrimSpace(trimmed[1:])
		shortcutMessage = "Enter a command to run."
		if value != "" {
			shortcutMessage = "Press Enter to run: " + value
		}
	}
	if shortcutMessage != "" {
		l.cancelSearch()
		l.clearItems()
		l.updateState(func(state *uiState) {
			state.resultCount = 0
			state.loading = false
		})
		l.message(shortcutMessage)
		return
	}
	l.message("")
	l.beginSearch(query)
}

func (l *launcher) showRecent() {
	items, _ := core.HandleTextInput("")
	l.mu.Lock()
	l.items = items
	l.mu.Unlock()
	l.updateState(func(state *uiState) {
		state.resultCount = len(items)
		state.selected = -1
		state.loading = false
	})
}

func (l *launcher) beginSearch(query string) {
	l.searchMu.Lock()
	if l.searchTimer != nil {
		l.searchTimer.Stop()
		l.searchTimer = nil
	}
	if l.searchCancel != nil {
		l.searchCancel()
		l.searchCancel = nil
	}
	l.searchGeneration++
	generation := l.searchGeneration
	ctx, cancel := context.WithCancel(context.Background())
	l.searchCancel = cancel
	l.searchTimer = time.AfterFunc(searchDebounce, func() {
		l.runSearch(ctx, generation, query)
	})
	l.searchMu.Unlock()
}

func (l *launcher) runSearch(ctx context.Context, generation uint64, query string) {
	l.searchMu.Lock()
	if generation != l.searchGeneration {
		l.searchMu.Unlock()
		return
	}
	l.searchTimer = nil
	l.searchMu.Unlock()

	items, message := core.HandleTextInput(query)
	if ctx.Err() != nil {
		return
	}

	l.searchMu.Lock()
	if generation != l.searchGeneration || ctx.Err() != nil {
		l.searchMu.Unlock()
		return
	}
	l.pendingSearch = &searchResult{generation: generation, query: query, items: items}
	if message != nil {
		l.pendingSearch.message = *message
	}
	l.searchMu.Unlock()
	l.window.Invalidate()
}

func (l *launcher) applyPendingSearch() {
	l.searchMu.Lock()
	result := l.pendingSearch
	l.pendingSearch = nil
	generation := l.searchGeneration
	l.searchMu.Unlock()
	if result == nil || result.generation != generation {
		return
	}

	state := l.snapshot()
	if state.page != pageLauncher || state.answer || state.query != result.query {
		return
	}
	if result.message != "" {
		l.clearItems()
		l.updateState(func(state *uiState) {
			state.resultCount = 0
			state.selected = -1
			state.loading = false
		})
		l.message(result.message)
		return
	}

	l.mu.Lock()
	l.items = result.items
	l.mu.Unlock()
	l.updateState(func(state *uiState) {
		state.resultCount = len(result.items)
		state.loading = false
		state.selected = -1
		if strings.TrimSpace(result.query) != "" && len(result.items) > 0 {
			state.selected = 0
		}
	})
	l.message("")
}

func (l *launcher) cancelSearch() {
	l.searchMu.Lock()
	if l.searchTimer != nil {
		l.searchTimer.Stop()
		l.searchTimer = nil
	}
	if l.searchCancel != nil {
		l.searchCancel()
		l.searchCancel = nil
	}
	l.searchGeneration++
	l.pendingSearch = nil
	l.searchMu.Unlock()
}

func (l *launcher) submit(gtx layout.Context, input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}
	switch input[0] {
	case '>':
		value := strings.TrimSpace(input[1:])
		if value == "" {
			l.message("Enter a command after >.")
			return
		}
		if err := utils.RunShellCommand(value); err != nil {
			l.message("Could not run command: " + err.Error())
		} else {
			HideWindow()
		}
		return
	}
	if strings.HasPrefix(input, ":") {
		l.editor.SetText("")
		if len(input) == 1 {
			l.message("Enter a command. Menu -> Help lists the available commands.")
			return
		}
		switch input[1] {
		case 'r':
			l.message("Re-indexing programs and documents.")
			go documents.SetupDocs()
			go apps.SetupApps()
		case 'q':
			HideWindow()
		case 'x':
			Quit()
		default:
			l.message("Unknown command. Menu -> Help lists the available commands.")
		}
		return
	}
	if strings.HasPrefix(input, "=") {
		expr := strings.ReplaceAll(strings.TrimPrefix(input, "="), " ", "")
		if utils.IsMath(expr) {
			if result, err := utils.EvalMath(expr); err == nil {
				l.editor.SetText(result)
				l.query(result)
				return
			}
		}
	}
	if selected := l.snapshot().selected; selected >= 0 {
		l.open(gtx, selected)
	}
}

func (l *launcher) askAssistant(prompt string) {
	l.cancelSearch()
	generation := l.answerGeneration.Add(1)
	l.clearItems()
	l.updateState(func(state *uiState) {
		state.answer = true
		state.loading = true
		state.resultCount = 0
		state.selected = -1
		state.answerPrompt = prompt
		state.answerCopied = false
		state.message = "Waiting for an answer…"
	})
	fetch := l.answerFetch
	if fetch == nil {
		fetch = utils.QuickAnswer
	}
	go func() {
		result := fetch(prompt)
		if strings.TrimSpace(result) == "" {
			result = "No answer returned. Try again."
		}
		l.updateState(func(state *uiState) {
			if l.answerGeneration.Load() != generation {
				return
			}
			state.loading = false
			state.message = result
		})
	}()
}

func (l *launcher) openWebSearch(query string) error {
	return utils.OpenURI(strings.ReplaceAll(g.SearchString, "%s", url.QueryEscape(query)))
}

func (l *launcher) selectResult(index int) {
	s := l.snapshot()
	if s.resultCount == 0 {
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= s.resultCount {
		index = s.resultCount - 1
	}
	if l.list.Position.Count > 0 {
		targetRow := index
		for rowIndex, row := range l.resultRows() {
			if !row.section && row.resourceIndex == index {
				targetRow = rowIndex
				break
			}
		}
		first := l.list.Position.First
		last := first + l.list.Position.Count - 1
		if l.list.Position.Count > 2 {
			last -= 2
		}
		switch {
		case targetRow < first:
			l.list.ScrollBy(float32(targetRow - first))
		case targetRow > last:
			l.list.ScrollBy(float32(targetRow - last))
		}
	}
	l.updateState(func(state *uiState) { state.selected = index })
}

func (l *launcher) open(gtx layout.Context, index int) {
	if item, ok := l.itemAt(index); ok {
		l.openResource(gtx, item)
	}
}

func (l *launcher) openResource(gtx layout.Context, item g.Resource) {
	if item.Assistant != "" {
		l.askAssistant(item.Assistant)
		return
	}
	if item.WebSearch != "" {
		if err := l.openWebSearch(item.WebSearch); err != nil {
			l.message("Sorry, there was an error opening your web browser.")
			return
		}
		HideWindow()
		return
	}
	if item.Command != nil {
		if systemactions.RequiresConfirmation(item.Command.Action) {
			l.pendingAction = item
			l.updateState(func(state *uiState) {
				state.page = pageConfirmation
				state.focusSearch = false
			})
			return
		}
		l.executeSystemAction(item)
		return
	}
	if item.Computed {
		l.copyText(gtx, item.Name)
		return
	}
	var err error
	if item.Document {
		err = documents.OpenFile(item.Filepath)
	} else {
		err = apps.OpenProgram(item.Filepath)
	}
	if err != nil {
		l.message("Sorry, there was an error opening the selected item.")
		return
	}
	if !item.Document {
		recent.RecordSelection(l.editor.Text(), item.Filepath)
	}
	HideWindow()
}

func (l *launcher) copyText(gtx layout.Context, value string) {
	if value != "" {
		gtx.Source.Execute(clipboard.WriteCmd{Type: "text/plain", Data: io.NopCloser(strings.NewReader(value))})
	}
}

func (l *launcher) executeSystemAction(item g.Resource) {
	if item.Command == nil || item.Command.Action == "" {
		return
	}
	l.pendingAction = g.Resource{}
	HideWindow()
	go func() {
		if err := systemactions.Execute(item.Command.Action); err != nil {
			log.Printf("system action %s failed: %v", item.Command.Action, err)
			ShowWindow()
			l.message("Could not run " + item.Name + ".")
			return
		}
	}()
}

func (l *launcher) itemAt(index int) (g.Resource, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if index < 0 || index >= len(l.items) {
		return g.Resource{}, false
	}
	return l.items[index], true
}

func (l *launcher) clearItems() {
	l.mu.Lock()
	l.items = nil
	l.mu.Unlock()
}

func (l *launcher) message(text string) {
	l.updateState(func(state *uiState) { state.message = text })
}

func (l *launcher) launcher() {
	if l.snapshot().page == pageSettings {
		l.commitIndexSettings(false)
	}
	l.pendingAction = g.Resource{}
	l.updateState(func(state *uiState) {
		state.page = pageLauncher
		state.focusSearch = true
	})
}

func (l *launcher) commitIndexSettings(force bool) bool {
	if !force && !l.indexSettingsPending() {
		return true
	}
	changed, err := documents.ApplyConfig(documents.IndexConfig{
		Roots:      documents.ParseIndexList(l.indexRoots.Text()),
		Exclusions: documents.ParseIndexList(l.indexExclusions.Text()),
	})
	if err != nil {
		l.settingsStatus = "Could not save index settings: " + err.Error()
		return false
	}
	l.savedIndexConfig = documents.Config()
	l.indexRoots.SetText(strings.Join(l.savedIndexConfig.Roots, "; "))
	l.indexExclusions.SetText(strings.Join(l.savedIndexConfig.Exclusions, "; "))
	if changed || force {
		l.settingsStatus = "Index settings saved; indexing…"
		go documents.SetupDocs()
	}
	return true
}

func (l *launcher) toggleTheme() {
	l.lightTheme = !l.lightTheme
	l.applyPalette()
	value := "dark"
	if l.lightTheme {
		value = "light"
	}
	if err := appsettings.SetSetting("theme", value); err != nil {
		l.settingsStatus = "Theme changed, but could not be saved: " + err.Error()
	} else {
		l.settingsStatus = "Theme saved."
	}
}

func (l *launcher) toggleDensity() {
	l.compact = !l.compact
	value := "comfortable"
	if l.compact {
		value = "compact"
	}
	if err := appsettings.SetSetting("density", value); err != nil {
		l.settingsStatus = "Density changed, but could not be saved: " + err.Error()
	} else {
		l.settingsStatus = "Density saved."
	}
}

func (l *launcher) layout(gtx layout.Context) layout.Dimensions {
	gtx.Metric.PxPerSp = effectiveSpScale(gtx.Metric.PxPerSp) * textSizeScale(l.textSizeIndex)
	paint.FillShape(gtx.Ops, l.palette.window, clip.Rect{Max: gtx.Constraints.Max}.Op())
	dimensions := layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions { return l.layoutPage(gtx) }),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if !l.snapshot().actionMenu {
				return layout.Dimensions{}
			}
			return l.resultActionMenu(gtx)
		}),
	)
	pass := pointer.PassOp{}.Push(gtx.Ops)
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	event.Op(gtx.Ops, &l.rootPointer)
	area.Pop()
	pass.Pop()
	return dimensions
}

func (l *launcher) layoutPage(gtx layout.Context) layout.Dimensions {
	s := l.snapshot()
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		switch s.page {
		case pageMenu:
			return l.menuPage(gtx)
		case pageHelp:
			return l.helpPage(gtx)
		case pageSettings:
			return l.settingsPage(gtx)
		case pageAbout:
			return l.textPage(gtx, "winfastnav", "Fast Windows navigation\n\nmarkski.ar\ngithub.com/markski1")
		case pageConfirmation:
			return l.confirmationPage(gtx)
		default:
			return l.launcherPage(gtx, s)
		}
	})
}

func (l *launcher) launcherPage(gtx layout.Context, s uiState) layout.Dimensions {
	for l.menu.Clicked(gtx) {
		l.updateState(func(state *uiState) {
			state.page = pageMenu
			state.focusSearch = false
		})
	}
	for l.clearSearch.Clicked(gtx) {
		l.editor.SetText("")
		l.query("")
	}
	editor := material.Editor(l.theme, &l.editor, "Search apps and documents...")
	editor.TextSize = unit.Sp(13)
	editor.Color = l.palette.text
	editor.HintColor = l.palette.muted
	queryHasText := strings.TrimSpace(l.editor.Text()) != ""
	resultGap := unit.Dp(10)
	if l.compact {
		resultGap = unit.Dp(6)
	}
	dimensions := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return l.input(gtx, editor.Layout) }),
				layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !queryHasText {
						return layout.Dimensions{}
					}
					return l.button(gtx, &l.clearSearch, "×")
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: resultGap}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return l.resultsPage(gtx, s) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.separator(gtx) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.statusBar(gtx, s) }),
	)
	if s.focusSearch {
		gtx.Source.Execute(key.FocusCmd{Tag: &l.editor})
		if gtx.Focused(&l.editor) {
			l.updateState(func(state *uiState) { state.focusSearch = false })
		} else {
			l.invalidate()
		}
	}
	return dimensions
}

func (l *launcher) resultsPage(gtx layout.Context, s uiState) layout.Dimensions {
	rows := l.resultRows()
	if len(rows) == 0 {
		return l.emptyResults(gtx, s)
	}
	return material.List(l.theme, &l.list).Layout(gtx, len(rows), func(gtx layout.Context, index int) layout.Dimensions {
		row := rows[index]
		if row.section {
			return l.resultSection(gtx, row.title)
		}
		l.updateResultActions(gtx, row.resourceIndex)
		for l.results[row.resourceIndex].Clicked(gtx) {
			if !l.snapshot().actionMenu {
				l.open(gtx, row.resourceIndex)
			}
		}
		return l.resultButton(gtx, &l.results[row.resourceIndex], row, row.resourceIndex == s.selected)
	})
}

func (l *launcher) resultRows() []resultRow {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var calculated, commandsRows, assistantRows, webSearchRows, applications, documentsRows []resultRow
	for resourceIndex, item := range l.items {
		if item.Computed {
			calculated = append(calculated, resultRow{title: item.Name, detail: "Calculated result / Enter to copy", kind: "Result", resourceIndex: resourceIndex})
			continue
		}
		if item.Command != nil {
			commandsRows = append(commandsRows, resultRow{title: item.Name, detail: item.Command.Detail, kind: "Command", resourceIndex: resourceIndex})
			continue
		}
		if item.WebSearch != "" {
			webSearchRows = append(webSearchRows, resultRow{title: item.Name, detail: "Open in browser", kind: "Web", resourceIndex: resourceIndex})
			continue
		}
		if item.Assistant != "" {
			assistantRows = append(assistantRows, resultRow{title: item.Name, detail: "Get a quick answer", kind: "Answer", resourceIndex: resourceIndex})
			continue
		}
		row := resultRow{title: item.Name, detail: filepath.Dir(item.Filepath), iconPath: item.Filepath, kind: "Application", resourceIndex: resourceIndex}
		if item.Document {
			row.kind = "Document"
			documentsRows = append(documentsRows, row)
			continue
		}
		applications = append(applications, row)
	}
	rows := append([]resultRow(nil), calculated...)
	if len(applications) > 0 {
		title := "APPS"
		state := l.snapshot()
		if strings.TrimSpace(state.query) == "" {
			title = "RECENT APPS"
		}
		rows = append(rows, resultRow{title: title, section: true})
		rows = append(rows, applications...)
	}
	if len(commandsRows) > 0 {
		rows = append(rows, resultRow{title: "COMMANDS", section: true})
		rows = append(rows, commandsRows...)
	}
	if len(assistantRows) > 0 {
		rows = append(rows, resultRow{title: "ASK", section: true})
		rows = append(rows, assistantRows...)
	}
	if len(documentsRows) > 0 {
		rows = append(rows, resultRow{title: "DOCUMENTS", section: true})
		rows = append(rows, documentsRows...)
	}
	if len(webSearchRows) > 0 {
		rows = append(rows, resultRow{title: "SEARCH", section: true})
		rows = append(rows, webSearchRows...)
	}
	return rows
}

func (l *launcher) textPage(gtx layout.Context, title, text string) layout.Dimensions {
	for l.back.Clicked(gtx) {
		l.backPage()
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.heading(gtx, title) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return l.scrollPage(gtx, func(gtx layout.Context) layout.Dimensions { return l.label(gtx, text) })
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.back, "Back") }),
	)
}

func (l *launcher) scrollPage(gtx layout.Context, widgets ...layout.Widget) layout.Dimensions {
	return material.List(l.theme, &l.pageList).LayoutWidgets(gtx, widgets...)
}

func (l *launcher) confirmationPage(gtx layout.Context) layout.Dimensions {
	for l.confirm.Clicked(gtx) {
		l.executeSystemAction(l.pendingAction)
	}
	for l.cancel.Clicked(gtx) {
		l.launcher()
	}
	title := l.pendingAction.Name
	if title == "" {
		l.launcher()
		return layout.Dimensions{}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.heading(gtx, "Confirm action") }),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return l.label(gtx, title+"? This action takes effect immediately.")
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.confirm, "Confirm") }),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.cancel, "Cancel") }),
			)
		}),
	)
}

func (l *launcher) button(gtx layout.Context, c *widget.Clickable, text string) layout.Dimensions {
	b := material.Button(l.theme, c, text)
	b.Background = l.palette.button
	b.Color = l.palette.buttonText
	b.CornerRadius = 0
	b.TextSize = unit.Sp(10.5)
	vertical := unit.Dp(6)
	if l.compact {
		vertical = unit.Dp(4)
	}
	b.Inset = layout.Inset{Top: vertical, Bottom: vertical, Left: unit.Dp(9), Right: unit.Dp(9)}
	return b.Layout(gtx)
}

func (l *launcher) resultButton(gtx layout.Context, c *widget.Clickable, row resultRow, selected bool) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		pass := pointer.PassOp{}.Push(gtx.Ops)
		event.Op(gtx.Ops, &l.resultContext[row.resourceIndex])
		pass.Pop()
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		background := l.palette.row
		if selected {
			background = l.palette.selected
		} else if c.Hovered() {
			background = l.palette.hover
		}
		return layout.Background{}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, background, clip.Rect{Max: gtx.Constraints.Min}.Op())
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}, func(gtx layout.Context) layout.Dimensions {
			vertical := unit.Dp(8)
			if l.compact {
				vertical = unit.Dp(5)
			}
			return layout.Inset{Top: vertical, Bottom: vertical, Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.resultIcon(gtx, row) }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return l.resultText(gtx, row.title, unit.Sp(13), l.palette.text)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return l.resultText(gtx, row.detail, unit.Sp(10.5), l.palette.secondary)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.resultMoreButton(gtx, row.resourceIndex) }),
				)
			})
		})
	})
}

func (l *launcher) resultIcon(gtx layout.Context, row resultRow) layout.Dimensions {
	size := gtx.Dp(unit.Dp(32))
	gtx.Constraints = layout.Exact(image.Pt(size, size))
	if row.iconPath != "" {
		if icon := l.icons.Image(row.iconPath); icon != nil {
			return widget.Image{Src: paint.NewImageOp(icon), Fit: widget.Contain}.Layout(gtx)
		}
	}
	paint.FillShape(gtx.Ops, l.palette.icon, clip.Rect{Max: gtx.Constraints.Min}.Op())
	letter := "•"
	if row.kind != "" {
		letter = strings.ToUpper(string([]rune(row.kind)[0]))
	}
	style := material.Label(l.theme, unit.Sp(12), letter)
	style.Color = l.palette.buttonText
	style.Alignment = text.Middle
	return layout.Center.Layout(gtx, style.Layout)
}

func (l *launcher) resultText(gtx layout.Context, value string, size unit.Sp, foreground color.NRGBA) layout.Dimensions {
	style := material.Label(l.theme, size, value)
	style.Color = foreground
	style.Alignment = text.Start
	style.MaxLines = 1
	style.Truncator = "…"
	return style.Layout(gtx)
}

func (l *launcher) resultSection(gtx layout.Context, title string) layout.Dimensions {
	style := material.Label(l.theme, unit.Sp(10.5), title)
	style.Color = l.palette.accent
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx, style.Layout)
}

func (l *launcher) emptyResults(gtx layout.Context, state uiState) layout.Dimensions {
	if state.loading {
		message := state.message
		if message == "" {
			message = "Working…"
		}
		if state.answer {
			return l.quickAnswer(gtx, message)
		}
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			style := material.Label(l.theme, unit.Sp(13), message)
			style.Color = l.palette.text
			return style.Layout(gtx)
		})
	}
	if state.answer {
		return l.quickAnswer(gtx, state.message)
	}
	if state.message == "" {
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}
	return l.centerMessage(gtx, state.message)
}

func (l *launcher) centerMessage(gtx layout.Context, message string) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		style := material.Label(l.theme, unit.Sp(13), message)
		style.Color = l.palette.text
		style.Alignment = text.Middle
		style.MaxLines = 3
		style.Truncator = "…"
		return style.Layout(gtx)
	})
}

func (l *launcher) statusBar(gtx layout.Context, state uiState) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if status := l.indexStatusTextLine(); status != "" {
					return l.statusText(gtx, status)
				}
				return l.keyboardHint(gtx, state)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.statusMenuButton(gtx) }),
		)
	})
}

func (l *launcher) indexStatusTextLine() string {
	documentStatus := documents.Status()
	appStatus := apps.Status()
	if documentStatus.Status == documents.IndexStatusReady && appStatus.Phase == apps.CatalogPhaseReady {
		return ""
	}
	var messages []string
	if documentStatus.Status != documents.IndexStatusReady {
		messages = append(messages, l.indexStatusText(documentStatus))
	}
	if appStatus.Phase != apps.CatalogPhaseReady {
		messages = append(messages, l.catalogStatusText(appStatus))
	}
	return strings.Join(messages, "   •   ")
}

func (l *launcher) indexStatusText(status documents.IndexSnapshot) string {
	textValue := "Documents: waiting for the first index"
	switch status.Status {
	case documents.IndexStatusIndexing:
		textValue = fmt.Sprintf("Indexing documents… %d found", status.ItemCount)
	case documents.IndexStatusStale:
		textValue = "Document index is out of date"
	case documents.IndexStatusError:
		textValue = "Document index error"
		if status.Error != "" {
			textValue += ": " + strings.Split(status.Error, "\n")[0]
		}
	case documents.IndexStatusReady:
		textValue = fmt.Sprintf("Documents ready: %d", status.ItemCount)
	}
	return textValue
}

func (l *launcher) catalogStatusText(status apps.CatalogSnapshot) string {
	textValue := "Applications: waiting for the first index"
	switch status.Phase {
	case apps.CatalogPhaseIndexing:
		textValue = fmt.Sprintf("Indexing applications… %d found", status.ItemCount)
	case apps.CatalogPhaseError:
		textValue = "Application index error"
		if status.Error != "" {
			textValue += ": " + strings.Split(status.Error, "\n")[0]
		}
	case apps.CatalogPhaseReady:
		textValue = fmt.Sprintf("Applications ready: %d", status.ItemCount)
	}
	return textValue
}

func (l *launcher) keyboardHint(gtx layout.Context, state uiState) layout.Dimensions {
	hint := "> command   Esc hide   Alt+Space summon"
	if state.answer {
		hint = "Esc hide   Alt+Space return to answer"
		if !state.loading {
			hint = "Ctrl+C copy answer   " + hint
		}
		return l.statusText(gtx, hint)
	}
	if item, ok := l.itemAt(state.selected); ok {
		var hints []string
		for _, choice := range actionsForResult(item) {
			switch choice.action {
			case resultOpen:
				hints = append(hints, "Enter open")
				if item.Command != nil {
					hints[len(hints)-1] = "Enter run"
				}
			case resultCopy:
				if item.Computed {
					hints = append(hints, "Enter copy")
				}
				hints = append(hints, "Ctrl+C copy")
			case resultReveal:
				hints = append(hints, "Ctrl+Enter reveal")
			case resultAdmin:
				hints = append(hints, "Shift+Enter admin")
			}
		}
		hint = strings.Join(hints, "   ")
	}
	return l.statusText(gtx, hint)
}

func (l *launcher) statusText(gtx layout.Context, value string) layout.Dimensions {
	style := material.Label(l.theme, unit.Sp(10), value)
	style.Color = l.palette.muted
	style.MaxLines = 1
	style.Truncator = "…"
	return style.Layout(gtx)
}

func (l *launcher) statusMenuButton(gtx layout.Context) layout.Dimensions {
	b := material.Button(l.theme, &l.menu, "Menu")
	b.Background = l.palette.button
	b.Color = l.palette.buttonText
	b.CornerRadius = 0
	b.TextSize = unit.Sp(9.5)
	b.Inset = layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}
	return b.Layout(gtx)
}

func (l *launcher) input(gtx layout.Context, content layout.Widget) layout.Dimensions {
	return l.surface(gtx, l.palette.input, inset(layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}, content))
}

func (l *launcher) separator(gtx layout.Context) layout.Dimensions {
	size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(1)))
	paint.FillShape(gtx.Ops, l.palette.surfaceEdge, clip.Rect{Max: size}.Op())
	return layout.Dimensions{Size: size}
}

func (l *launcher) label(gtx layout.Context, text string) layout.Dimensions {
	s := material.Body1(l.theme, text)
	s.Color = l.palette.text
	return s.Layout(gtx)
}

func (l *launcher) heading(gtx layout.Context, text string) layout.Dimensions {
	s := material.H6(l.theme, text)
	s.Color = l.palette.text
	return s.Layout(gtx)
}
