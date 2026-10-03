package ui

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func inset(padding layout.Inset, content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions { return padding.Layout(gtx, content) }
}

func column(gap unit.Dp, widgets ...layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(widgets)*2)
		for index, widget := range widgets {
			if index > 0 && gap > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Height: gap}.Layout))
			}
			children = append(children, layout.Rigid(widget))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}
}

func (l *launcher) title(value string, size unit.Sp) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		style := material.Label(l.theme, size, value)
		style.Font.Weight, style.Color = font.Medium, l.palette.text
		return style.Layout(gtx)
	}
}

func (l *launcher) description(value string) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		style := material.Label(l.theme, unit.Sp(11), value)
		style.Color, style.LineHeightScale = l.palette.secondary, 1.3
		return style.Layout(gtx)
	}
}

func (l *launcher) surface(gtx layout.Context, background color.NRGBA, content layout.Widget) layout.Dimensions {
	return layout.Background{}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, background, clip.Rect{Max: gtx.Constraints.Min}.Op())
		return layout.Dimensions{Size: gtx.Constraints.Min}
	}, content)
}

func (l *launcher) pageHeader(gtx layout.Context, title string) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, l.title(title, unit.Sp(18))),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, &l.back, "Back") }),
		)
	})
}

func (l *launcher) panePage(gtx layout.Context, title string, sidebar layout.Widget, list *widget.List, footer string, widgets ...layout.Widget) layout.Dimensions {
	for l.back.Clicked(gtx) {
		l.backPage()
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageHeader(gtx, title) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{}.Layout(gtx,
				layout.Rigid(sidebar),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return l.surface(gtx, l.palette.surface, inset(layout.UniformInset(unit.Dp(16)), func(gtx layout.Context) layout.Dimensions {
						return material.List(l.theme, list).LayoutWidgets(gtx, widgets...)
					}))
				}),
			)
		}),
		layout.Rigid(inset(layout.Inset{Top: unit.Dp(12)}, l.description(footer))),
	)
}

func (l *launcher) paneSidebar(gtx layout.Context, titles []string, buttons []widget.Clickable, selected int) layout.Dimensions {
	width := min(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(116)))
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = width, width
	widgets := make([]layout.Widget, len(titles))
	for index, title := range titles {
		button := &buttons[index]
		widgets[index] = buttonWidget(button, func(gtx layout.Context) layout.Dimensions {
			background := l.palette.window
			if selected == index {
				background = l.palette.selected
			} else if button.Hovered() || gtx.Focused(button) {
				background = l.palette.hover
			}
			return l.surface(gtx, background, inset(layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(10), Right: unit.Dp(6)}, func(gtx layout.Context) layout.Dimensions {
				style := material.Label(l.theme, unit.Sp(12), title)
				style.Color = l.palette.secondary
				if selected == index {
					style.Color, style.Font.Weight = l.palette.text, font.Medium
				}
				return style.Layout(gtx)
			}))
		})
	}
	return column(unit.Dp(4), widgets...)(gtx)
}

func buttonWidget(button *widget.Clickable, content layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions { return button.Layout(gtx, content) }
}

func (l *launcher) choiceGroup(buttons []widget.Clickable, labels []string, selected int) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(labels)*2)
		for index, label := range labels {
			if index > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout))
			}
			children = append(children, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				style := material.Button(l.theme, &buttons[index], label)
				style.TextSize, style.CornerRadius = unit.Sp(12), 0
				style.Color, style.Background = l.palette.text, l.palette.input
				style.Inset = layout.UniformInset(unit.Dp(10))
				if index == selected {
					style.Background, style.Font.Weight = l.palette.selected, font.Medium
				}
				return style.Layout(gtx)
			}))
		}
		return layout.Flex{}.Layout(gtx, children...)
	}
}
