package ui

import (
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type helpPane uint8

const (
	helpShortcuts helpPane = iota
	helpSearch
	helpTools
	helpCommands
	helpPaneCount
)

var helpPaneTitles = [helpPaneCount]string{"Shortcuts", "Search", "Tools", "Commands"}

type helpEntry struct {
	example     string
	description string
}

func (l *launcher) helpPage(gtx layout.Context) layout.Dimensions {
	for pane := range l.helpNav {
		for l.helpNav[pane].Clicked(gtx) {
			if l.helpPane != helpPane(pane) {
				l.helpPane = helpPane(pane)
				l.helpList.Position = layout.Position{}
			}
		}
	}
	for l.back.Clicked(gtx) {
		l.backPage()
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageHeader(gtx, "Help") }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return l.paneSidebar(gtx, helpPaneTitles[:], l.helpNav[:], int(l.helpPane))
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return l.pageSurface(gtx, l.palette.surface, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return material.List(l.theme, &l.helpList).LayoutWidgets(gtx, l.helpPaneWidgets()...)
						})
					})
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return l.pageDescription(gtx, "Esc returns to Menu.")
			})
		}),
	)
}

func (l *launcher) helpPaneWidgets() []layout.Widget {
	var title, description string
	var entries []helpEntry
	var notes []helpEntry
	switch l.helpPane {
	case helpSearch:
		title, description = "Search tips", "Type an app or document name to find it."
		entries = []helpEntry{
			{"pdf report", "Find PDFs matching “report”."},
			{"type:docx budget", "Find Word documents matching “budget”."},
			{"folder:work invoice", "Find invoices in folders matching “work”."},
		}
		notes = []helpEntry{
			{"Web search and quick answers", "When nothing matches, choose Ask the assistant or Search the web. Answers have Copy and Retry controls and stay available when you hide the launcher."},
			{"Manage your search", "Choose document folders and your web search provider in Settings. Hidden apps can be restored there too."},
		}
	case helpTools:
		title, description = "Everyday tools", "Type these directly in the search box."
		entries = []helpEntry{
			{"(2+3)^2", "Calculate an expression."},
			{"20% of 80", "Work out a percentage."},
			{"10 km to mi", "Convert between units."},
			{"100 USD to EUR", "Convert currency."},
			{"today", "Show the current date."},
			{"in 2 weeks", "Find a future date."},
		}
		notes = []helpEntry{{"Copy a result", "Select a calculated result and press Enter to copy it."}}
	case helpCommands:
		title, description = "Commands", "Type a command in search, then press Enter."
		entries = []helpEntry{
			{"> command", "Run a command with cmd.exe."},
			{":r", "Re-index apps and documents."},
			{":q", "Hide the launcher."},
			{":x", "Quit winfastnav."},
		}
		notes = []helpEntry{{"Windows actions", "Search for lock, sleep, restart, or Windows Settings to see available actions."}}
	default:
		title, description = "Keyboard shortcuts", "Right-click a result or choose … for actions. Shift+F10 opens actions from the keyboard."
		entries = []helpEntry{
			{"Alt + Space", "Show the launcher."},
			{"Up / Down", "Select a search result."},
			{"Enter", "Open or run the selected result."},
			{"Ctrl + Enter", "Reveal the result in Explorer."},
			{"Shift + Enter", "Run an app as administrator."},
			{"Ctrl + C", "Copy the selected result."},
			{"Delete", "Hide an app from search."},
			{"Esc", "Hide the launcher or leave a page."},
		}
	}
	widgets := []layout.Widget{func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageTitle(gtx, title, unit.Sp(16)) }),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageDescription(gtx, description) }),
			)
		})
	}}
	for _, entry := range entries {
		widgets = append(widgets, func(gtx layout.Context) layout.Dimensions { return l.helpRow(gtx, entry) })
	}
	for _, note := range notes {
		widgets = append(widgets, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageTitle(gtx, note.example, unit.Sp(12)) }),
					layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageDescription(gtx, note.description) }),
				)
			})
		})
	}
	return widgets
}

func (l *launcher) helpRow(gtx layout.Context, entry helpEntry) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				width := min(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(128)))
				gtx.Constraints.Min.X, gtx.Constraints.Max.X = width, width
				return l.pageSurface(gtx, l.palette.input, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return l.pageTitle(gtx, entry.example, unit.Sp(11))
					})
				})
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return l.pageDescription(gtx, entry.description) }),
		)
	})
}
