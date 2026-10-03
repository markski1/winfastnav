package ui

import (
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"winfastnav/internal/documents"
	g "winfastnav/internal/globals"
	"winfastnav/internal/utils"
)

func (l *launcher) menuPage(gtx layout.Context) layout.Dimensions {
	for l.help.Clicked(gtx) {
		l.updateState(func(state *uiState) {
			state.page = pageHelp
			state.focusSearch = false
		})
	}
	for l.settingsButton.Clicked(gtx) {
		l.settings.SetText(g.SearchString)
		config := documents.Config()
		l.savedIndexConfig = config
		l.indexRoots.SetText(strings.Join(config.Roots, "; "))
		l.indexExclusions.SetText(strings.Join(config.Exclusions, "; "))
		l.startupEnabled = utils.IsInStartup()
		l.startupSwitch.Value = l.startupEnabled
		l.settingsStatus = ""
		l.updateState(func(state *uiState) {
			state.page = pageSettings
			state.focusSearch = false
		})
	}
	for l.about.Clicked(gtx) {
		l.pageList.Position = layout.Position{}
		l.updateState(func(state *uiState) {
			state.page = pageAbout
			state.focusSearch = false
		})
	}
	for l.quit.Clicked(gtx) {
		Quit()
	}
	for l.back.Clicked(gtx) {
		l.launcher()
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageHeader(gtx, "Menu") }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return l.menuRow(gtx, &l.settingsButton, "Settings", "Search, appearance, indexing, and hidden apps.", "›")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return l.menuRow(gtx, &l.help, "Help", "Keyboard shortcuts, search tips, and commands.", "›")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return l.menuRow(gtx, &l.about, "About", "App information and project links.", "›")
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return l.menuRow(gtx, &l.quit, "Quit winfastnav", "Close the app and stop the background launcher.", "×")
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
		layout.Rigid(l.description("Esc returns to search.")),
	)
}

func (l *launcher) menuRow(gtx layout.Context, button *widget.Clickable, title, description, marker string) layout.Dimensions {
	return inset(layout.Inset{Bottom: unit.Dp(8)}, buttonWidget(button, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		background := l.palette.surface
		if button.Hovered() || gtx.Focused(button) {
			background = l.palette.hover
		}
		return l.surface(gtx, background, inset(layout.UniformInset(unit.Dp(14)), func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, column(unit.Dp(5), l.title(title, unit.Sp(14)), l.description(description))),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					style := material.Label(l.theme, unit.Sp(20), marker)
					style.Color = l.palette.secondary
					return style.Layout(gtx)
				}),
			)
		}))
	}))(gtx)
}
