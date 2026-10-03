package ui

import (
	"image/color"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
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
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageDescription(gtx, "Esc returns to search.") }),
	)
}

func (l *launcher) pageHeader(gtx layout.Context, title string) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return l.pageTitle(gtx, title, unit.Sp(18)) }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.back, "Back") }),
		)
	})
}

func (l *launcher) menuRow(gtx layout.Context, button *widget.Clickable, title, description, marker string) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return button.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			background := l.palette.surface
			if button.Hovered() || gtx.Focused(button) {
				background = l.palette.hover
			}
			return l.pageSurface(gtx, background, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(14)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageTitle(gtx, title, unit.Sp(14)) }),
								layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageDescription(gtx, description) }),
							)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							style := material.Label(l.theme, unit.Sp(20), marker)
							style.Color = l.palette.secondary
							return style.Layout(gtx)
						}),
					)
				})
			})
		})
	})
}

func (l *launcher) paneSidebar(gtx layout.Context, titles []string, buttons []widget.Clickable, selected int) layout.Dimensions {
	width := min(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(116)))
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = width, width
	children := make([]layout.FlexChild, 0, len(titles))
	for index, title := range titles {
		button := &buttons[index]
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return button.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					background := l.palette.window
					if selected == index {
						background = l.palette.selected
					} else if button.Hovered() || gtx.Focused(button) {
						background = l.palette.hover
					}
					return l.pageSurface(gtx, background, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(10), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							style := material.Label(l.theme, unit.Sp(12), title)
							style.Color = l.palette.secondary
							if selected == index {
								style.Color = l.palette.text
								style.Font.Weight = font.Medium
							}
							return style.Layout(gtx)
						})
					})
				})
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (l *launcher) pageTitle(gtx layout.Context, title string, size unit.Sp) layout.Dimensions {
	style := material.Label(l.theme, size, title)
	style.Font.Weight = font.Medium
	style.Color = l.palette.text
	return style.Layout(gtx)
}

func (l *launcher) pageDescription(gtx layout.Context, value string) layout.Dimensions {
	style := material.Label(l.theme, unit.Sp(11), value)
	style.Color = l.palette.secondary
	style.LineHeightScale = 1.3
	return style.Layout(gtx)
}

func (l *launcher) pageSurface(gtx layout.Context, background color.NRGBA, content layout.Widget) layout.Dimensions {
	return layout.Background{}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, background, clip.Rect{Max: gtx.Constraints.Min}.Op())
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}, content)
}
