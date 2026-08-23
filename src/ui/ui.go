package ui

import (
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

	"gioui.org/app"
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
	"winfastnav/internal/utils"
	"winfastnav/internal/windowcontrol"
)

const maxResults = 30

type launcher struct {
	controller                                                *presentation.Controller
	windowControl                                             *windowcontrol.Controller
	window                                                    app.Window
	ops                                                       op.Ops
	theme                                                     *material.Theme
	icons                                                     *appicons.Cache
	editor, settings                                          widget.Editor
	list                                                      widget.List
	results                                                   [maxResults]widget.Clickable
	menu, modeButton, back, help, settingsButton, about, quit widget.Clickable
	startup, clear, confirm, cancel                           widget.Clickable
	mu                                                        sync.RWMutex
	items                                                     []g.Resource
	confirmClear                                              bool
	startupEnabled                                            bool
	settingsStatus                                            string
	centered                                                  bool
}

var active *launcher

type resultRow struct {
	title    string
	detail   string
	iconPath string
	kind     string
}

func SetupUI() {
	theme := material.NewTheme()
	theme.TextSize = unit.Sp(12.35)
	active = &launcher{controller: presentation.NewController(g.ModeSearchProgram), theme: theme, icons: appicons.NewCache(), list: widget.List{List: layout.List{Axis: layout.Vertical}}}
	active.editor.SingleLine, active.editor.Submit = true, true
	active.window.Option(app.Title(g.AppName), app.Size(unit.Dp(580), unit.Dp(460)), app.MinSize(unit.Dp(580), unit.Dp(460)), app.MaxSize(unit.Dp(580), unit.Dp(460)), app.Decorated(false), app.TopMost(true))
	active.windowControl = windowcontrol.New(g.AppName)
	active.controller.Post(presentation.Command{Kind: presentation.CommandShow})
	active.message(g.AppName + "\nMenu -> Help")
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
	g.CurrentMode = g.ModeSearchProgram
	active.clearItems()
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandShow})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetMode, Mode: g.ModeSearchProgram})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageLauncher})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetQuery})
	active.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetResults})
	active.query("")
	active.message(g.AppName + "\nMenu -> Help")
	_ = active.windowControl.ShowAndFocus()
	active.window.Invalidate()
}

func HideWindow() {
	if active == nil {
		return
	}
	active.clearItems()
	active.controller.Post(presentation.Command{Kind: presentation.CommandSetQuery})
	active.controller.Post(presentation.Command{Kind: presentation.CommandSetResults})
	active.controller.Post(presentation.Command{Kind: presentation.CommandHide})
	go func() {
		_ = active.windowControl.Hide()
	}()
}

func ShowAbout() {
	if active != nil {
		active.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageAbout})
	}
}
func Quit() {
	if active != nil {
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
		case app.ViewEvent:
			l.windowControl.BindView(e)
		case app.FrameEvent:
			gtx := app.NewContext(&l.ops, e)
			l.controller.SetInvalidator(l.window.Invalidate)
			l.update(gtx)
			l.layout(gtx)
			e.Frame(&l.ops)
		}
	}
}

func (l *launcher) update(gtx layout.Context) {
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
	if (event.Name == key.NameReturn || event.Name == key.NameEnter) && event.Modifiers.Contain(key.ModAlt) {
		l.revealSelected()
		return
	}
	if event.Name == "C" && event.Modifiers.Contain(key.ModCtrl|key.ModShift) {
		l.copySelectedPath(gtx)
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
	l.controller.Post(presentation.Command{Kind: presentation.CommandSetQuery, Query: query})
	items, message := core.HandleTextInput(query)
	if message != nil {
		l.clearItems()
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetResults})
		l.message(*message)
		return
	}
	if g.CurrentMode == g.ModeSearchProgram || g.CurrentMode == g.ModeSearchDocument {
		l.mu.Lock()
		l.items = items
		l.mu.Unlock()
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetResults, ResultCount: len(items)})
		l.message("")
	}
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
		case 'd':
			l.mode(g.ModeSearchDocument)
		case 'w':
			l.mode(g.ModeSearchInternet)
		case 'g':
			l.mode(g.ModeAskGPT)
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
	case g.ModeAskGPT:
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetLoading, Loading: true})
		l.message("Please wait...")
		go func(p string) {
			result := utils.MakeGPTReq(p)
			l.controller.Post(presentation.Command{Kind: presentation.CommandSetLoading, Loading: false})
			l.message(result)
		}(input)
	case g.ModeSearchInternet:
		if err := utils.OpenURI(strings.ReplaceAll(g.SearchString, "%s", url.QueryEscape(input))); err != nil {
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

func (l *launcher) mode(mode int) {
	g.CurrentMode = mode
	l.clearItems()
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetMode, Mode: mode})
	l.controller.Dispatch(presentation.Command{Kind: presentation.CommandSetResults})
	l.message("")
	l.query(l.editor.Text())
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
		first := l.list.Position.First
		last := first + l.list.Position.Count - 1
		if l.list.Position.Count > 2 {
			last -= 2
		}
		switch {
		case index < first:
			l.list.ScrollBy(float32(index - first))
		case index > last:
			l.list.ScrollBy(float32(index - last))
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
	if item.Computed {
		l.editor.SetText(item.Name)
		l.query(item.Name)
		return
	}
	var err error
	if g.CurrentMode == g.ModeSearchProgram {
		err = apps.OpenProgram(item.Filepath)
	} else if g.CurrentMode == g.ModeSearchDocument {
		err = documents.OpenFile(item.Filepath)
	}
	if err != nil {
		l.message("Sorry, there was an error opening the selected item.")
		return
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
	if item.Computed {
		return
	}
	apps.BlockApplication(item)
	l.query(l.editor.Text())
}

func (l *launcher) cycleMode() {
	switch g.CurrentMode {
	case g.ModeSearchProgram:
		l.mode(g.ModeSearchDocument)
	case g.ModeSearchDocument:
		l.mode(g.ModeSearchInternet)
	default:
		l.mode(g.ModeSearchProgram)
	}
}

func (l *launcher) selectedPath(index int) string {
	if index < 0 || (g.CurrentMode != g.ModeSearchProgram && g.CurrentMode != g.ModeSearchDocument) {
		return ""
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if index >= len(l.items) {
		return ""
	}
	return l.items[index].Filepath
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

func (l *launcher) copySelectedPath(gtx layout.Context) {
	path := l.selectedPath(l.controller.Snapshot().Selected)
	if path == "" {
		return
	}
	gtx.Source.Execute(clipboard.WriteCmd{Type: "text/plain", Data: io.NopCloser(strings.NewReader(path))})
}

func (l *launcher) clearItems() { l.mu.Lock(); l.items = nil; l.mu.Unlock() }
func (l *launcher) message(text string) {
	l.controller.Post(presentation.Command{Kind: presentation.CommandSetMessage, Message: utils.WrapTextByWords(text, 64)})
}
func (l *launcher) launcher() {
	l.confirmClear = false
	l.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageLauncher})
}

func (l *launcher) layout(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0x1a, G: 0x18, B: 0x18, A: 0xff}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	s := l.controller.Snapshot()
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		switch s.Page {
		case presentation.PageMenu:
			return l.menuPage(gtx)
		case presentation.PageHelp:
			return l.textPage(gtx, "Help", "ALT + SPACE: Summon\nESC: Hide\nDelete: Hide app\n\n:p Program search\n:d Document search\n:w Internet search\n:g Quick GPT\n:r Re-index\n:x Quit\n\nType 2+2 or 20in for calculations and conversions.")
		case presentation.PageSettings:
			return l.settingsPage(gtx)
		case presentation.PageAbout:
			return l.textPage(gtx, "winfastnav", "Fast Windows navigation\n\nmarkski.ar\ngithub.com/markski1")
		default:
			return l.launcherPage(gtx, s)
		}
	})
}

func (l *launcher) launcherPage(gtx layout.Context, s presentation.State) layout.Dimensions {
	for l.menu.Clicked(gtx) {
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageMenu})
	}
	for l.modeButton.Clicked(gtx) {
		l.cycleMode()
	}
	hint := placeholder(s.Mode)
	if s.Mode == g.ModeSearchDocument && !g.FinishedCachingDocs {
		hint = "Document search [still caching]..."
	}
	editor := material.Editor(l.theme, &l.editor, hint)
	editor.TextSize = unit.Sp(13)
	editor.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	editor.HintColor = color.NRGBA{R: 180, G: 180, B: 180, A: 255}
	dimensions := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return l.input(gtx, editor.Layout) }),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.modeButton, modeTitle(s.Mode)) }),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.menu, "Menu") }),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.modeStatus(gtx, s) }),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
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
		for l.results[index].Clicked(gtx) {
			l.open(index)
		}
		return l.resultButton(gtx, &l.results[index], rows[index], index == s.Selected)
	})
}

func (l *launcher) resultRows() []resultRow {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var rows []resultRow
	for _, item := range l.items {
		if item.Computed {
			rows = append(rows, resultRow{title: item.Name, detail: "Calculated result / Enter to use", kind: "Result"})
			continue
		}
		kind := "Application"
		if g.CurrentMode == g.ModeSearchDocument {
			kind = "Document"
		}
		rows = append(rows, resultRow{title: item.Name, detail: filepath.Dir(item.Filepath), iconPath: item.Filepath, kind: kind})
	}
	return rows
}

func (l *launcher) menuPage(gtx layout.Context) layout.Dimensions {
	for l.help.Clicked(gtx) {
		l.controller.Post(presentation.Command{Kind: presentation.CommandSetPage, Page: presentation.PageHelp})
	}
	for l.settingsButton.Clicked(gtx) {
		l.settings.SetText(g.SearchString)
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
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.heading(gtx, title) }), layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return l.label(gtx, text) }), layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.back, "Back") }))
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
	for l.back.Clicked(gtx) {
		l.launcher()
	}
	editor := material.Editor(l.theme, &l.settings, "https://duckduckgo.com/?q=%s")
	editor.TextSize = unit.Sp(13)
	editor.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	editor.HintColor = color.NRGBA{R: 180, G: 180, B: 180, A: 255}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.heading(gtx, "Settings") }),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.section(gtx, "SEARCH") }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return l.settingNote(gtx, "URL template. %s is replaced with the query.")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.input(gtx, editor.Layout) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.settingNote(gtx, l.settingsStatus) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.separator(gtx) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.section(gtx, "STARTUP") }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := "Enable launch at sign-in"
			if l.startupEnabled {
				label = "Launch at sign-in: enabled"
			}
			return l.menuButton(gtx, &l.startup, label)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.separator(gtx) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.section(gtx, "HIDDEN APPS") }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return l.settingNote(gtx, "Hidden apps are excluded from app search.")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return l.menuButton(gtx, &l.clear, fmt.Sprintf("Restore hidden apps (%d)", len(g.ExecBlocklist)))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !l.confirmClear {
				return layout.Dimensions{}
			}
			return layout.Flex{}.Layout(gtx, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.confirm, "Restore apps") }), layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.menuButton(gtx, &l.cancel, "Keep hidden") }))
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{} }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.back, "Back") }),
	)
}
func (l *launcher) button(gtx layout.Context, c *widget.Clickable, text string) layout.Dimensions {
	b := material.Button(l.theme, c, text)
	b.Background = color.NRGBA{R: 0x46, G: 0x38, B: 0x38, A: 255}
	b.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	b.CornerRadius = 0
	b.TextSize = unit.Sp(10.5)
	b.Inset = layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}
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
		background := color.NRGBA{R: 0x21, G: 0x1e, B: 0x1e, A: 0xff}
		if selected {
			background = color.NRGBA{R: 0x64, G: 0x4c, B: 0x4c, A: 0xff}
		} else if c.Hovered() {
			background = color.NRGBA{R: 0x36, G: 0x2d, B: 0x2d, A: 0xff}
		}
		return layout.Background{}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, background, clip.Rect{Max: gtx.Constraints.Min}.Op())
			return layout.Dimensions{Size: gtx.Constraints.Min}
		}, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.resultIcon(gtx, row) }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return l.resultText(gtx, row.title, unit.Sp(13), color.NRGBA{R: 255, G: 255, B: 255, A: 255})
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return l.resultText(gtx, row.detail, unit.Sp(10.5), color.NRGBA{R: 0xb8, G: 0xb2, B: 0xb2, A: 255})
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
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0x58, G: 0x46, B: 0x46, A: 0xff}, clip.Rect{Max: gtx.Constraints.Min}.Op())
	letter := "•"
	if row.kind != "" {
		letter = strings.ToUpper(string([]rune(row.kind)[0]))
	}
	style := material.Label(l.theme, unit.Sp(12), letter)
	style.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	style.Alignment = text.Middle
	return style.Layout(gtx)
}

func (l *launcher) resultText(gtx layout.Context, value string, size unit.Sp, foreground color.NRGBA) layout.Dimensions {
	style := material.Label(l.theme, size, value)
	style.Color = foreground
	style.Alignment = text.Start
	style.MaxLines = 1
	style.Truncator = "…"
	return style.Layout(gtx)
}

func (l *launcher) modeStatus(gtx layout.Context, state presentation.State) layout.Dimensions {
	label := "Click Apps to switch modes. :p apps  :d documents  :w web"
	if state.Mode == g.ModeSearchDocument && !g.FinishedCachingDocs {
		label = "Documents are still indexing. Search will improve as indexing completes."
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			style := material.Label(l.theme, unit.Sp(10.5), strings.ToUpper(modeTitle(state.Mode)))
			style.Color = color.NRGBA{R: 0xd3, G: 0xaf, B: 0xaf, A: 255}
			return style.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			style := material.Label(l.theme, unit.Sp(10.5), label)
			style.Color = color.NRGBA{R: 0xa8, G: 0xa2, B: 0xa2, A: 255}
			style.MaxLines = 1
			style.Truncator = "…"
			return style.Layout(gtx)
		}),
	)
}

func (l *launcher) emptyResults(gtx layout.Context, state presentation.State) layout.Dimensions {
	if state.Message == "" {
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		style := material.Label(l.theme, unit.Sp(13), state.Message)
		style.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		style.Alignment = text.Middle
		style.MaxLines = 3
		style.Truncator = "…"
		return style.Layout(gtx)
	})
}

func (l *launcher) keyboardHint(gtx layout.Context, state presentation.State) layout.Dimensions {
	hint := "↑ ↓ move   Enter open   Esc hide   Alt+Space summon"
	if state.Selected >= 0 && l.selectedPath(state.Selected) != "" {
		hint = "Alt+Enter show in folder   Ctrl+Shift+C copy path   " + hint
	}
	return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		style := material.Label(l.theme, unit.Sp(10), hint)
		style.Color = color.NRGBA{R: 0xa8, G: 0xa2, B: 0xa2, A: 255}
		style.MaxLines = 1
		style.Truncator = "…"
		return style.Layout(gtx)
	})
}

func (l *launcher) input(gtx layout.Context, content layout.Widget) layout.Dimensions {
	return layout.Background{}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, color.NRGBA{R: 0x2b, G: 0x2b, B: 0x2b, A: 0xff}, clip.Rect{Max: gtx.Constraints.Min}.Op())
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, content)
	})
}

func (l *launcher) section(gtx layout.Context, text string) layout.Dimensions {
	style := material.Body1(l.theme, text)
	style.Color = color.NRGBA{R: 0xc8, G: 0xc8, B: 0xc8, A: 0xff}
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(3)}.Layout(gtx, style.Layout)
}

func (l *launcher) settingNote(gtx layout.Context, value string) layout.Dimensions {
	if value == "" {
		return layout.Dimensions{}
	}
	style := material.Label(l.theme, unit.Sp(10.5), value)
	style.Color = color.NRGBA{R: 0xb8, G: 0xb2, B: 0xb2, A: 255}
	style.MaxLines = 1
	style.Truncator = "…"
	return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, style.Layout)
}

func (l *launcher) separator(gtx layout.Context) layout.Dimensions {
	size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(1)))
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0x4a, G: 0x4a, B: 0x4a, A: 0xff}, clip.Rect{Max: size}.Op())
	return layout.Dimensions{Size: size}
}
func (l *launcher) label(gtx layout.Context, text string) layout.Dimensions {
	s := material.Body1(l.theme, text)
	s.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	return s.Layout(gtx)
}
func (l *launcher) heading(gtx layout.Context, text string) layout.Dimensions {
	s := material.H6(l.theme, text)
	s.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	return s.Layout(gtx)
}
func placeholder(mode int) string {
	switch mode {
	case g.ModeSearchDocument:
		return "Document search..."
	case g.ModeSearchInternet:
		return "Internet search..."
	case g.ModeAskGPT:
		return "Quick GPT..."
	default:
		return "Program search..."
	}
}

func modeTitle(mode int) string {
	switch mode {
	case g.ModeSearchDocument:
		return "Documents"
	case g.ModeSearchInternet:
		return "Web"
	case g.ModeAskGPT:
		return "Quick GPT"
	default:
		return "Apps"
	}
}
