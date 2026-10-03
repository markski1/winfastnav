package ui

import (
	"gioui.org/layout"
	"gioui.org/unit"
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
	sidebar := func(gtx layout.Context) layout.Dimensions {
		return l.paneSidebar(gtx, helpPaneTitles[:], l.helpNav[:], int(l.helpPane))
	}
	return l.panePage(gtx, "Help", sidebar, &l.helpList, "Esc returns to Menu.", l.helpPaneWidgets()...)
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
	widgets := []layout.Widget{inset(layout.Inset{Bottom: unit.Dp(16)}, column(unit.Dp(6), l.title(title, unit.Sp(16)), l.description(description)))}
	for _, entry := range entries {
		widgets = append(widgets, func(gtx layout.Context) layout.Dimensions { return l.helpRow(gtx, entry) })
	}
	for _, note := range notes {
		widgets = append(widgets, inset(layout.Inset{Top: unit.Dp(14)}, column(unit.Dp(6), l.title(note.example, unit.Sp(12)), l.description(note.description))))
	}
	return widgets
}

func (l *launcher) helpRow(gtx layout.Context, entry helpEntry) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				width := min(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(128)))
				gtx.Constraints.Min.X, gtx.Constraints.Max.X = width, width
				return l.surface(gtx, l.palette.input, inset(layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}, l.title(entry.example, unit.Sp(11))))
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Flexed(1, l.description(entry.description)),
		)
	})
}
