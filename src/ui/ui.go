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
	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/getlantern/systray"
	"winfastnav/internal/apps"
	"winfastnav/internal/core"
	"winfastnav/internal/documents"
	g "winfastnav/internal/globals"
	appicons "winfastnav/internal/icons"
	"winfastnav/internal/presentation"
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
	controller                                    *presentation.Controller
	windowControl                                 *windowcontrol.Controller
	window                                        app.Window
	ops                                           op.Ops
	theme                                         *material.Theme
	icons                                         *appicons.Cache
	editor, settings, aliases                     widget.Editor
	indexRoots, indexExclusions                   widget.Editor
	list                                          widget.List
	pageList                                      widget.List
	results                                       [maxResults + 2]widget.Clickable
	menu, back, help, settingsButton, about, quit widget.Clickable
	startup, clear, confirm, cancel               widget.Clickable
	clearSearch, themeToggle, densityToggle       widget.Clickable
	reindex                                       widget.Clickable
	mu                                            sync.RWMutex
	items                                         []g.Resource
	confirmClear                                  bool
	startupEnabled                                bool
	settingsStatus                                string
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
	pendingAction                                 g.Resource
	searchMu                                      sync.Mutex
	searchGeneration                              uint64
	searchCancel                                  context.CancelFunc
	searchTimer                                   *time.Timer
	pendingSearch                                 *searchResult
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
	mode       int
	query      string
	items      []g.Resource
	message    string
}

type answerAtom struct {
	text string
	span markdownSpan
	link int
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
	themeSetting, _ := appsettings.GetSetting("theme")
	densitySetting, _ := appsettings.GetSetting("density")
	active = &launcher{
		controller: presentation.NewController(g.ModeSearchProgram),
		theme:      theme,
		icons:      appicons.NewCache(),
		list:       widget.List{List: layout.List{Axis: layout.Vertical}},
		pageList:   widget.List{List: layout.List{Axis: layout.Vertical}},
		lightTheme: strings.EqualFold(themeSetting, "light"),
		compact:    strings.EqualFold(densitySetting, "compact"),
	}
	active.icons.SetChangedHandler(active.invalidate)
	active.applyPalette()
	active.editor.SingleLine, active.editor.Submit = true, true
	active.aliases.SingleLine = true
	active.indexRoots.SingleLine, active.indexExclusions.SingleLine = true, true
	active.window.Option(app.Title(g.AppName), app.Size(unit.Dp(580), unit.Dp(460)), app.MinSize(unit.Dp(580), unit.Dp(460)), app.MaxSize(unit.Dp(580), unit.Dp(460)), app.Decorated(false), app.TopMost(true))
	active.windowControl = windowcontrol.New(g.AppName)
	active.controller.Post(presentation.Command{Kind: presentation.CommandHide})
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
	go func() {
		if err := active.run(); err != nil {
			log.Printf("Gio window closed: %v", err)
		}
	}()
}

func ShowWindow() {
	if active == nil {
		return
	}
	generation := active.windowGeneration.Add(1)
	if active.controller.Snapshot().Page == presentation.PageSettings {
		core.UpdateAliasSetting(g.AliasString)
		active.commitIndexSettings()
	}
	g.CurrentMode = g.ModeSearchProgram
	active.clearItems()
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandShow})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetMode, Mode: g.ModeSearchProgram})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageLauncher})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetQuery})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetResults})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetMessage})
	active.cancelSearch()
	active.showRecent()
	active.showAndFocus(generation)
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandFocusSearch})
	active.invalidate()
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
		if generation == l.windowGeneration.Load() && l.controller.Snapshot().Visible {
			log.Printf("failed to show launcher: %v", err)
		}
	})
}

func (l *launcher) setWindowVisible(generation uint64, visible bool) error {
	l.windowMu.Lock()
	defer l.windowMu.Unlock()
	if generation != l.windowGeneration.Load() || l.controller.Snapshot().Visible != visible {
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
	if active.controller.Snapshot().Visible {
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
	active.answerGeneration.Add(1)
	active.cancelSearch()
	active.clearItems()
	active.focused.Store(false)
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetQuery})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetResults})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetLoading, Loading: false})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetMessage})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandHide})
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

func (l *launcher) invalidate() {
	if l.windowReady.Load() {
		l.window.Invalidate()
	}
}

func ShowAbout() {
	if active != nil {
		active.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageAbout})
	}
}
func Quit() {
	if active != nil {
		if active.controller.Snapshot().Page == presentation.PageSettings {
			core.UpdateAliasSetting(g.AliasString)
			active.commitIndexSettings()
		}
		active.cancelSearch()
		active.controller.Close()
	}
	systray.Quit()
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
			} else if l.focused.Swap(false) && l.controller.Snapshot().Visible {
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
			l.controller.SetInvalidator(l.window.Invalidate)
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
	if err := l.setWindowVisible(generation, false); err != nil {
		log.Printf("failed to hide launcher on startup: %v", err)
	}
}

func (l *launcher) update(gtx layout.Context) {
	if l.refreshPending.Swap(false) {
		state := l.controller.Snapshot()
		if state.Mode == g.ModeSearchProgram && state.Page == presentation.PageLauncher {
			l.query(state.Query)
		}
	}
	l.applyPendingSearch()
	state := l.controller.Snapshot()
	if l.editor.Text() != state.Query {
		l.editor.SetText(state.Query)
	}
	for {
		e, ok := gtx.Source.Event(
			key.Filter{Name: key.NameUpArrow},
			key.Filter{Name: key.NameDownArrow},
			key.Filter{Name: key.NameReturn},
			key.Filter{Name: key.NameEnter},
			key.Filter{Name: key.NameEscape},
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
		)
		if !ok {
			break
		}
		if k, ok := e.(key.Event); ok && k.State == key.Press {
			l.key(gtx, k)
		}
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
			l.submit(l.editor.Text())
		}
	}
}

func (l *launcher) key(gtx layout.Context, event key.Event) {
	s := l.controller.Snapshot()
	if s.Page == presentation.PageConfirmation {
		switch event.Name {
		case key.NameEscape:
			l.launcher()
		case key.NameReturn, key.NameEnter:
			l.executeSystemAction(l.pendingAction)
		}
		return
	}
	if (event.Name == key.NameReturn || event.Name == key.NameEnter) && event.Modifiers.Contain(key.ModShift) {
		l.runSelectedElevated()
		return
	}
	if (event.Name == key.NameReturn || event.Name == key.NameEnter) && event.Modifiers.Contain(key.ModCtrl) {
		l.revealSelected()
		return
	}
	if (event.Name == key.NameReturn || event.Name == key.NameEnter) && event.Modifiers.Contain(key.ModAlt) {
		l.revealSelected()
		return
	}
	if event.Name == "C" && event.Modifiers.Contain(key.ModCtrl) {
		l.copySelected(gtx)
		return
	}

	switch event.Name {
	case key.NameEscape:
		if s.Page == presentation.PageLauncher {
			HideWindow()
		} else {
			l.launcher()
		}
	case key.NameUpArrow:
		l.selectResult(s.Selected - 1)
	case key.NameDownArrow:
		l.selectResult(s.Selected + 1)
	case key.NameReturn, key.NameEnter:
		if s.Selected >= 0 {
			l.open(s.Selected)
		} else if g.CurrentMode != g.ModeSearchProgram {
			l.submit(l.editor.Text())
		}
	case key.NameDeleteForward:
		if g.CurrentMode == g.ModeSearchProgram && s.Selected >= 0 {
			l.block(s.Selected)
		}
	case key.NameHome:
		l.selectResult(0)
	case key.NameEnd:
		l.selectResult(s.ResultCount - 1)
	case key.NamePageUp:
		l.selectResult(s.Selected - max(l.list.Position.Count-2, 1))
	case key.NamePageDown:
		l.selectResult(s.Selected + max(l.list.Position.Count-2, 1))
	}
}

func (l *launcher) query(query string) {
	if query == ":a" {
		l.activateCommandMode(g.ModeQuickAnswer)
		return
	}
	if query == ":w" {
		l.activateCommandMode(g.ModeSearchInternet)
		return
	}
	mode := l.controller.Snapshot().Mode
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetQuery, Query: query})
	if mode == g.ModeQuickAnswer {
		return
	}
	l.message("")
	l.beginSearch(query, mode)
}

func (l *launcher) showRecent() {
	items, _ := core.HandleTextInputMode("", g.ModeSearchProgram)
	l.mu.Lock()
	l.items = items
	l.mu.Unlock()
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetResults, ResultCount: len(items)})
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetLoading, Loading: false})
}

func (l *launcher) beginSearch(query string, mode int) {
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
		l.runSearch(ctx, generation, query, mode)
	})
	l.searchMu.Unlock()
}

func (l *launcher) runSearch(ctx context.Context, generation uint64, query string, mode int) {
	l.searchMu.Lock()
	if generation != l.searchGeneration {
		l.searchMu.Unlock()
		return
	}
	l.searchTimer = nil
	l.searchMu.Unlock()

	items, message := core.HandleTextInputMode(query, mode)
	if ctx.Err() != nil {
		return
	}

	l.searchMu.Lock()
	if generation != l.searchGeneration || ctx.Err() != nil {
		l.searchMu.Unlock()
		return
	}
	l.pendingSearch = &searchResult{generation: generation, mode: mode, query: query, items: items}
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

	state := l.controller.Snapshot()
	if state.Page != presentation.PageLauncher || state.Mode != result.mode || state.Query != result.query {
		return
	}
	if result.message != "" {
		l.clearItems()
		l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetResults})
		l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetLoading, Loading: false})
		l.message(result.message)
		return
	}

	l.mu.Lock()
	l.items = result.items
	l.mu.Unlock()
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetResults, ResultCount: len(result.items)})
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetLoading, Loading: false})
	if firstResultSelected(result.query, len(result.items)) {
		l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSelectResult, Selected: 0})
	}
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

func firstResultSelected(query string, resultCount int) bool {
	return strings.TrimSpace(query) != "" && resultCount > 0
}

func (l *launcher) submit(input string) {
	if input == "" {
		return
	}
	if strings.HasPrefix(input, ":") {
		l.editor.SetText("")
		if len(input) == 1 {
			l.message("Enter a command. Menu -> Help lists the available commands.")
			return
		}
		switch input[1] {
		case 'p':
			l.mode(g.ModeSearchProgram)
		case 'w':
			l.mode(g.ModeSearchInternet)
		case 'a':
			l.mode(g.ModeQuickAnswer)
		case 'r':
			l.message("Re-indexing programs and documents.")
			go documents.SetupDocs()
			go apps.SetupApps()
		case 'q':
			HideWindow()
		case 'x':
			Quit()
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
	switch g.CurrentMode {
	case g.ModeQuickAnswer:
		generation := l.answerGeneration.Add(1)
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetLoading, Loading: true})
		l.message("Waiting for an answer...")
		go func(p string) {
			result := utils.QuickAnswer(p)
			if l.answerGeneration.Load() != generation {
				return
			}
			l.controller.Post(presentation.Command{Kind: presentation.CommandSetLoading, Loading: false})
			l.message(result)
		}(input)
	case g.ModeSearchInternet:
		if err := l.openWebSearch(input); err != nil {
			l.message("Sorry, there was an error opening your web browser.")
		} else {
			HideWindow()
		}
	default:
		if s := l.controller.Snapshot(); s.Selected >= 0 {
			l.open(s.Selected)
		}
	}
}

func (l *launcher) openWebSearch(query string) error {
	return utils.OpenURI(strings.ReplaceAll(g.SearchString, "%s", url.QueryEscape(query)))
}

func (l *launcher) mode(mode int) {
	l.answerGeneration.Add(1)
	g.CurrentMode = mode
	l.clearItems()
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetMode, Mode: mode})
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetResults})
	l.message("")
	l.query(l.editor.Text())
}

func (l *launcher) activateCommandMode(mode int) {
	l.answerGeneration.Add(1)
	l.cancelSearch()
	g.CurrentMode = mode
	l.clearItems()
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetMode, Mode: mode})
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetQuery})
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetResults})
	l.message("")
	l.editor.SetText("")
}
func (l *launcher) selectResult(index int) {
	s := l.controller.Snapshot()
	if s.ResultCount == 0 {
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= s.ResultCount {
		index = s.ResultCount - 1
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
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSelectResult, Selected: index})
}
func (l *launcher) open(index int) {
	l.mu.RLock()
	if index >= len(l.items) {
		l.mu.RUnlock()
		return
	}
	item := l.items[index]
	l.mu.RUnlock()
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
			l.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageConfirmation})
			return
		}
		l.executeSystemAction(item)
		return
	}
	if item.Computed {
		l.editor.SetText(item.Name)
		l.query(item.Name)
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
func (l *launcher) block(index int) {
	l.mu.RLock()
	if index >= len(l.items) {
		l.mu.RUnlock()
		return
	}
	item := l.items[index]
	l.mu.RUnlock()
	if item.Computed || item.Document || item.Command != nil || item.WebSearch != "" {
		return
	}
	apps.BlockApplication(item)
	l.query(l.editor.Text())
}

func (l *launcher) selectedPath(index int) string {
	if index < 0 || g.CurrentMode != g.ModeSearchProgram {
		return ""
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if index >= len(l.items) {
		return ""
	}
	path := l.items[index].Filepath
	if !filepath.IsAbs(path) {
		return ""
	}
	return path
}

func (l *launcher) revealSelected() {
	path := l.selectedPath(l.controller.Snapshot().Selected)
	if path == "" {
		return
	}
	if err := utils.RevealInFolder(path); err != nil {
		return
	}
}

func (l *launcher) copySelected(gtx layout.Context) {
	item, ok := l.selectedItem()
	if !ok || item.Command != nil {
		return
	}
	value := item.Filepath
	if item.Computed {
		value = item.Name
	}
	if value != "" {
		gtx.Source.Execute(clipboard.WriteCmd{Type: "text/plain", Data: io.NopCloser(strings.NewReader(value))})
	}
}

func (l *launcher) runSelectedElevated() {
	item, ok := l.selectedItem()
	if !ok || item.Computed || item.Document || item.Command != nil || item.WebSearch != "" {
		return
	}
	if err := apps.RunProgramElevated(item.Filepath); err != nil {
		l.message("The selected application cannot be run as administrator.")
		return
	}
	recent.RecordSelection(l.editor.Text(), item.Filepath)
	HideWindow()
}

func (l *launcher) executeSystemAction(item g.Resource) {
	if item.Command == nil || item.Command.Action == "" {
		return
	}
	query := l.controller.Snapshot().Query
	l.pendingAction = g.Resource{}
	HideWindow()
	go func() {
		if err := systemactions.Execute(item.Command.Action); err != nil {
			log.Printf("system action %s failed: %v", item.Command.Action, err)
			ShowWindow()
			l.message("Could not run " + item.Name + ".")
			return
		}
		recent.Record(item.Filepath)
		recent.RecordSelection(query, item.Filepath)
	}()
}

func (l *launcher) selectedItem() (g.Resource, bool) {
	index := l.controller.Snapshot().Selected
	l.mu.RLock()
	defer l.mu.RUnlock()
	if index < 0 || index >= len(l.items) {
		return g.Resource{}, false
	}
	return l.items[index], true
}

func (l *launcher) clearItems() { l.mu.Lock(); l.items = nil; l.mu.Unlock() }
func (l *launcher) message(text string) {
	l.controller.Post(presentation.Command{Kind: presentation.CommandSetMessage, Message: utils.WrapTextByWords(text, 64)})
}
func (l *launcher) launcher() {
	if l.controller.Snapshot().Page == presentation.PageSettings {
		core.UpdateAliasSetting(l.aliases.Text())
		l.commitIndexSettings()
	}
	l.confirmClear = false
	l.pendingAction = g.Resource{}
	l.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageLauncher})
}

func (l *launcher) commitIndexSettings() {
	changed, err := documents.ApplyConfig(documents.IndexConfig{
		Roots:      documents.ParseIndexList(l.indexRoots.Text()),
		Exclusions: documents.ParseIndexList(l.indexExclusions.Text()),
	})
	if err != nil {
		l.settingsStatus = "Could not save index settings: " + err.Error()
		return
	}
	if changed {
		l.settingsStatus = "Index settings saved; indexing…"
		go documents.SetupDocs()
	}
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
	paint.FillShape(gtx.Ops, l.palette.window, clip.Rect{Max: gtx.Constraints.Max}.Op())
	s := l.controller.Snapshot()
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		switch s.Page {
		case presentation.PageMenu:
			return l.menuPage(gtx)
		case presentation.PageHelp:
			return l.textPage(gtx, "Help", "ALT + SPACE: Summon\nESC: Hide\nENTER: Open or run\nCTRL + ENTER: Reveal in Explorer\nSHIFT + ENTER: Run as administrator\nCTRL + C: Copy selected result\nDELETE: Hide app\n\n:w Internet search\n:a Quick Answer\n:r Re-index\n:x Quit\n\nDocuments: pdf report, type:docx, folder:work\nAliases: configure alias=application in Settings\n\nTry (2+3)^2, 20% of 80, 10 km to mi, or 100 USD to EUR.")
		case presentation.PageSettings:
			return l.settingsPage(gtx)
		case presentation.PageAbout:
			return l.textPage(gtx, "winfastnav", "Fast Windows navigation\n\nmarkski.ar\ngithub.com/markski1")
		case presentation.PageConfirmation:
			return l.confirmationPage(gtx)
		default:
			return l.launcherPage(gtx, s)
		}
	})
}

func (l *launcher) launcherPage(gtx layout.Context, s presentation.State) layout.Dimensions {
	for l.menu.Clicked(gtx) {
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageMenu})
	}
	for l.clearSearch.Clicked(gtx) {
		l.editor.SetText("")
		l.query("")
	}
	hint := placeholder(s.Mode)
	editor := material.Editor(l.theme, &l.editor, hint)
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
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.menu, "Menu") }),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.indexStatusLine(gtx) }),
		layout.Rigid(layout.Spacer{Height: resultGap}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return l.resultsPage(gtx, s) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.separator(gtx) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.keyboardHint(gtx, s) }),
	)
	if s.FocusSearch {
		gtx.Source.Execute(key.FocusCmd{Tag: &l.editor})
		if gtx.Focused(&l.editor) {
			l.controller.Post(presentation.Command{Kind: presentation.CommandFocusHandled})
		} else {
			l.window.Invalidate()
		}
	}
	return dimensions
}

func (l *launcher) resultsPage(gtx layout.Context, s presentation.State) layout.Dimensions {
	rows := l.resultRows()
	if len(rows) == 0 {
		return l.emptyResults(gtx, s)
	}
	return material.List(l.theme, &l.list).Layout(gtx, len(rows), func(gtx layout.Context, index int) layout.Dimensions {
		row := rows[index]
		if row.section {
			return l.resultSection(gtx, row.title)
		}
		for l.results[row.resourceIndex].Clicked(gtx) {
			l.open(row.resourceIndex)
		}
		return l.resultButton(gtx, &l.results[row.resourceIndex], row, row.resourceIndex == s.Selected)
	})
}

func (l *launcher) resultRows() []resultRow {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var calculated, commandsRows, webSearchRows, applications, documentsRows []resultRow
	for resourceIndex, item := range l.items {
		if item.Computed {
			calculated = append(calculated, resultRow{title: item.Name, detail: "Calculated result / Enter to use", kind: "Result", resourceIndex: resourceIndex})
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
		rows = append(rows, resultRow{title: "APPS", section: true})
		rows = append(rows, applications...)
	}
	if len(commandsRows) > 0 {
		rows = append(rows, resultRow{title: "COMMANDS", section: true})
		rows = append(rows, commandsRows...)
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

func (l *launcher) menuPage(gtx layout.Context) layout.Dimensions {
	for l.help.Clicked(gtx) {
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageHelp})
	}
	for l.settingsButton.Clicked(gtx) {
		l.settings.SetText(g.SearchString)
		l.aliases.SetText(g.AliasString)
		config := documents.Config()
		l.indexRoots.SetText(strings.Join(config.Roots, "; "))
		l.indexExclusions.SetText(strings.Join(config.Exclusions, "; "))
		l.startupEnabled = utils.IsInStartup()
		l.settingsStatus = "Changes are saved automatically."
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageSettings})
	}
	for l.about.Clicked(gtx) {
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageAbout})
	}
	for l.quit.Clicked(gtx) {
		Quit()
	}
	for l.back.Clicked(gtx) {
		l.launcher()
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.help, "Help") }), layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.settingsButton, "Settings") }), layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.about, "About") }), layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.quit, "Quit") }), layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }), layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.back, "Back") }))
}

func (l *launcher) textPage(gtx layout.Context, title, text string) layout.Dimensions {
	for l.back.Clicked(gtx) {
		l.launcher()
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

func (l *launcher) settingsPage(gtx layout.Context) layout.Dimensions {
	for {
		e, ok := l.settings.Update(gtx)
		if !ok {
			break
		}
		if _, changed := e.(widget.ChangeEvent); changed {
			core.UpdateSearchSetting(l.settings.Text())
			l.settingsStatus = "Search URL saved."
		}
	}
	for {
		e, ok := l.aliases.Update(gtx)
		if !ok {
			break
		}
		if _, changed := e.(widget.ChangeEvent); changed {
			g.AliasString = l.aliases.Text()
			apps.SetAliases(g.AliasString)
			l.settingsStatus = "Aliases will be saved when you leave Settings."
		}
	}
	for {
		e, ok := l.indexRoots.Update(gtx)
		if !ok {
			break
		}
		if _, changed := e.(widget.ChangeEvent); changed {
			l.settingsStatus = "Index settings will be saved when you leave Settings."
		}
	}
	for {
		e, ok := l.indexExclusions.Update(gtx)
		if !ok {
			break
		}
		if _, changed := e.(widget.ChangeEvent); changed {
			l.settingsStatus = "Index settings will be saved when you leave Settings."
		}
	}
	for l.startup.Clicked(gtx) {
		if err := utils.AddToStartup(); err != nil {
			l.settingsStatus = "Could not enable startup: " + err.Error()
		} else {
			l.startupEnabled = true
			l.settingsStatus = "Startup enabled."
		}
	}
	for l.clear.Clicked(gtx) {
		l.confirmClear = true
	}
	for l.confirm.Clicked(gtx) {
		apps.UnblockAllApplications()
		l.confirmClear = false
		l.settingsStatus = "Hidden apps restored."
	}
	for l.cancel.Clicked(gtx) {
		l.confirmClear = false
	}
	for l.themeToggle.Clicked(gtx) {
		l.toggleTheme()
	}
	for l.densityToggle.Clicked(gtx) {
		l.toggleDensity()
	}
	for l.reindex.Clicked(gtx) {
		l.commitIndexSettings()
		l.settingsStatus = "Indexing documents…"
		go documents.SetupDocs()
	}
	for l.back.Clicked(gtx) {
		l.launcher()
	}
	editor := material.Editor(l.theme, &l.settings, "https://duckduckgo.com/?q=%s")
	editor.TextSize = unit.Sp(13)
	editor.Color = l.palette.text
	editor.HintColor = l.palette.muted
	aliasEditor := material.Editor(l.theme, &l.aliases, "vsc=Visual Studio Code; dc=Discord")
	aliasEditor.TextSize = unit.Sp(13)
	aliasEditor.Color = l.palette.text
	aliasEditor.HintColor = l.palette.muted
	rootsEditor := material.Editor(l.theme, &l.indexRoots, `%USERPROFILE%\Documents; D:\Projects`)
	rootsEditor.TextSize = unit.Sp(13)
	rootsEditor.Color = l.palette.text
	rootsEditor.HintColor = l.palette.muted
	exclusionsEditor := material.Editor(l.theme, &l.indexExclusions, `node_modules; venv; C:\Temp\Archive`)
	exclusionsEditor.TextSize = unit.Sp(13)
	exclusionsEditor.Color = l.palette.text
	exclusionsEditor.HintColor = l.palette.muted
	indexStatus := documents.Status()
	themeLabel := "Theme: Dark (switch to light)"
	if l.lightTheme {
		themeLabel = "Theme: Light (switch to dark)"
	}
	densityLabel := "Density: Comfortable (switch to compact)"
	if l.compact {
		densityLabel = "Density: Compact (switch to comfortable)"
	}
	widgets := []layout.Widget{
		func(gtx layout.Context) layout.Dimensions { return l.section(gtx, "SEARCH") },
		func(gtx layout.Context) layout.Dimensions {
			return l.settingNote(gtx, "URL template. %s is replaced with the query.")
		},
		func(gtx layout.Context) layout.Dimensions { return l.input(gtx, editor.Layout) },
		func(gtx layout.Context) layout.Dimensions { return l.settingNote(gtx, l.settingsStatus) },
		func(gtx layout.Context) layout.Dimensions { return l.separator(gtx) },
		func(gtx layout.Context) layout.Dimensions { return l.section(gtx, "APP ALIASES") },
		func(gtx layout.Context) layout.Dimensions {
			return l.settingNote(gtx, "Separate aliases with semicolons: alias=application name")
		},
		func(gtx layout.Context) layout.Dimensions { return l.input(gtx, aliasEditor.Layout) },
		func(gtx layout.Context) layout.Dimensions { return l.separator(gtx) },
		func(gtx layout.Context) layout.Dimensions { return l.section(gtx, "INDEXING") },
		func(gtx layout.Context) layout.Dimensions {
			return l.settingNote(gtx, "Folders to search, separated by semicolons. Changes apply when you leave Settings.")
		},
		func(gtx layout.Context) layout.Dimensions { return l.input(gtx, rootsEditor.Layout) },
		func(gtx layout.Context) layout.Dimensions {
			return l.settingNote(gtx, "Excluded folder names or absolute paths, separated by semicolons.")
		},
		func(gtx layout.Context) layout.Dimensions { return l.input(gtx, exclusionsEditor.Layout) },
		func(gtx layout.Context) layout.Dimensions { return l.indexStatusNote(gtx, indexStatus) },
		func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.reindex, "Re-index now") },
		func(gtx layout.Context) layout.Dimensions { return l.separator(gtx) },
		func(gtx layout.Context) layout.Dimensions { return l.section(gtx, "APPEARANCE") },
		func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.themeToggle, themeLabel) },
		func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.densityToggle, densityLabel) },
		func(gtx layout.Context) layout.Dimensions { return l.separator(gtx) },
		func(gtx layout.Context) layout.Dimensions { return l.section(gtx, "STARTUP") },
		func(gtx layout.Context) layout.Dimensions {
			label := "Enable launch at sign-in"
			if l.startupEnabled {
				label = "Launch at sign-in: enabled"
			}
			return l.menuButton(gtx, &l.startup, label)
		},
		func(gtx layout.Context) layout.Dimensions { return l.separator(gtx) },
		func(gtx layout.Context) layout.Dimensions { return l.section(gtx, "HIDDEN APPS") },
		func(gtx layout.Context) layout.Dimensions {
			return l.settingNote(gtx, "Hidden apps are excluded from app search.")
		},
		func(gtx layout.Context) layout.Dimensions {
			return l.menuButton(gtx, &l.clear, fmt.Sprintf("Restore hidden apps (%d)", len(g.ExecBlocklist)))
		},
		func(gtx layout.Context) layout.Dimensions {
			if !l.confirmClear {
				return layout.Dimensions{}
			}
			return layout.Flex{}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.confirm, "Restore apps") }), layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.cancel, "Keep hidden") }))
		},
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.heading(gtx, "Settings") }),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return l.scrollPage(gtx, widgets...) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.back, "Back") }),
	)
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

func (l *launcher) menuButton(gtx layout.Context, c *widget.Clickable, text string) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return l.button(gtx, c, text)
	})
}

func (l *launcher) resultButton(gtx layout.Context, c *widget.Clickable, row resultRow, selected bool) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
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

func (l *launcher) emptyResults(gtx layout.Context, state presentation.State) layout.Dimensions {
	if state.Loading {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			message := state.Message
			if message == "" {
				message = "Working…"
			}
			style := material.Label(l.theme, unit.Sp(13), message)
			style.Color = l.palette.text
			return style.Layout(gtx)
		})
	}
	if state.Message == "" {
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}
	if state.Mode == g.ModeQuickAnswer {
		return l.quickAnswer(gtx, state.Message)
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		style := material.Label(l.theme, unit.Sp(13), state.Message)
		style.Color = l.palette.text
		style.Alignment = text.Middle
		style.MaxLines = 3
		style.Truncator = "…"
		return style.Layout(gtx)
	})
}

func (l *launcher) quickAnswer(gtx layout.Context, message string) layout.Dimensions {
	if l.answerMessage != message {
		l.answerMessage = message
		l.answerSpans = parseBasicMarkdown(message)
		l.answerURLs = l.answerURLs[:0]
		l.answerLinks = [maxAnswerLinks]widget.Clickable{}
		for index := range l.answerSpans {
			if l.answerSpans[index].url == "" || len(l.answerURLs) == maxAnswerLinks {
				continue
			}
			l.answerURLs = append(l.answerURLs, l.answerSpans[index].url)
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

	lines := l.answerLines(gtx)
	return material.List(l.theme, &l.pageList).Layout(gtx, len(lines), func(gtx layout.Context, index int) layout.Dimensions {
		line := lines[index]
		return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, len(line))
			for _, atom := range line {
				atom := atom
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return l.answerAtom(gtx, atom)
				}))
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
		})
	})
}

func (l *launcher) answerLines(gtx layout.Context) [][]answerAtom {
	var lines [][]answerAtom
	var line []answerAtom
	width := 0
	linkIndex := 0
	for _, span := range l.answerSpans {
		if span.url != "" {
			link := -1
			if linkIndex < len(l.answerURLs) {
				link = linkIndex
			}
			l.addAnswerAtom(gtx, &lines, &line, &width, answerAtom{text: span.text, span: span, link: link})
			linkIndex++
			continue
		}
		for _, text := range splitMarkdownText(span.text) {
			if text == "\n" && len(line) > 0 {
				lines = append(lines, line)
				line, width = nil, 0
				continue
			}
			l.addAnswerAtom(gtx, &lines, &line, &width, answerAtom{text: text, span: span, link: -1})
		}
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}
	return lines
}

func (l *launcher) addAnswerAtom(gtx layout.Context, lines *[][]answerAtom, line *[]answerAtom, width *int, atom answerAtom) {
	measure := gtx
	measure.Constraints.Min.X = 0
	measure.Constraints.Max.X = 1 << 20
	recording := op.Record(gtx.Ops)
	dimensions := l.answerAtom(measure, atom)
	recording.Stop()
	if *width > 0 && *width+dimensions.Size.X > gtx.Constraints.Max.X {
		*lines = append(*lines, *line)
		*line, *width = nil, 0
	}
	if *width == 0 && strings.TrimSpace(atom.text) == "" {
		return
	}
	*line = append(*line, atom)
	*width += dimensions.Size.X
}

func (l *launcher) answerAtom(gtx layout.Context, atom answerAtom) layout.Dimensions {
	style := material.Label(l.theme, unit.Sp(13), atom.text)
	style.Color = l.palette.text
	if atom.span.bold {
		style.Font.Weight = font.Bold
	}
	if atom.span.italic {
		style.Font.Style = font.Italic
	}
	if atom.link >= 0 {
		style.Color = l.palette.accent
		return l.answerLinks[atom.link].Layout(gtx, style.Layout)
	}
	return style.Layout(gtx)
}

func (l *launcher) indexStatusLine(gtx layout.Context) layout.Dimensions {
	documentStatus := documents.Status()
	appStatus := apps.Status()
	if documentStatus.Status == documents.IndexStatusReady && appStatus.Phase == apps.CatalogPhaseReady {
		return layout.Dimensions{}
	}
	var messages []string
	if documentStatus.Status != documents.IndexStatusReady {
		messages = append(messages, l.indexStatusText(documentStatus))
	}
	if appStatus.Phase != apps.CatalogPhaseReady {
		messages = append(messages, l.catalogStatusText(appStatus))
	}
	return l.settingNote(gtx, strings.Join(messages, "   •   "))
}

func (l *launcher) indexStatusNote(gtx layout.Context, status documents.IndexSnapshot) layout.Dimensions {
	return l.settingNote(gtx, l.indexStatusText(status))
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

func (l *launcher) keyboardHint(gtx layout.Context, state presentation.State) layout.Dimensions {
	hint := "↑ ↓ move   Enter open   Esc hide   Alt+Space summon"
	if item, ok := l.selectedItem(); ok {
		hint = "Enter open   Ctrl+C copy"
		if item.Command != nil {
			hint = "Enter run"
		}
		if item.Computed {
			hint = "Enter use   Ctrl+C copy"
		}
		if l.selectedPath(state.Selected) != "" {
			hint += "   Ctrl+Enter reveal"
		}
		if !item.Computed && !item.Document && filepath.IsAbs(item.Filepath) {
			hint += "   Shift+Enter admin"
		}
	}
	return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		style := material.Label(l.theme, unit.Sp(10), hint)
		style.Color = l.palette.muted
		style.MaxLines = 1
		style.Truncator = "…"
		return style.Layout(gtx)
	})
}

func (l *launcher) input(gtx layout.Context, content layout.Widget) layout.Dimensions {
	return layout.Background{}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, l.palette.input, clip.Rect{Max: gtx.Constraints.Min}.Op())
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, content)
	})
}

func (l *launcher) section(gtx layout.Context, text string) layout.Dimensions {
	style := material.Body1(l.theme, text)
	style.Color = l.palette.secondary
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(3)}.Layout(gtx, style.Layout)
}

func (l *launcher) settingNote(gtx layout.Context, value string) layout.Dimensions {
	if value == "" {
		return layout.Dimensions{}
	}
	style := material.Label(l.theme, unit.Sp(10.5), value)
	style.Color = l.palette.secondary
	style.MaxLines = 1
	style.Truncator = "…"
	return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, style.Layout)
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
func placeholder(mode int) string {
	switch mode {
	case g.ModeSearchInternet:
		return "Internet search..."
	case g.ModeQuickAnswer:
		return "Quick Answer..."
	default:
		return "Search apps and documents..."
	}
}
