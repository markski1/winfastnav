package ui

import (
	"image"
	"image/color"
	"path/filepath"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"winfastnav/internal/apps"
	g "winfastnav/internal/globals"
	"winfastnav/internal/utils"
)

type resultAction uint8

const (
	resultOpen resultAction = iota
	resultReveal
	resultCopy
	resultAdmin
	resultHide
)

type resultActionChoice struct {
	action   resultAction
	label    string
	shortcut string
}

func actionsForResult(item g.Resource) []resultActionChoice {
	if item.Computed {
		return []resultActionChoice{{resultCopy, "Copy result", "Ctrl+C"}}
	}
	label := "Open"
	if item.Assistant != "" {
		label = "Ask assistant"
	}
	if item.WebSearch != "" {
		label = "Search the web"
	}
	if item.Command != nil {
		label = "Run action"
	}
	choices := []resultActionChoice{{resultOpen, label, "Enter"}}
	if !filepath.IsAbs(item.Filepath) || item.Command != nil || item.Assistant != "" || item.WebSearch != "" {
		return choices
	}
	choices = append(choices, resultActionChoice{resultReveal, "Reveal in Explorer", "Ctrl+Enter"}, resultActionChoice{resultCopy, "Copy path", "Ctrl+C"})
	if !item.Document {
		choices = append(choices, resultActionChoice{resultAdmin, "Run as administrator", "Shift+Enter"}, resultActionChoice{resultHide, "Hide from search", "Delete"})
	}
	return choices
}

func (l *launcher) openResultActions(index int) {
	l.mu.RLock()
	if index < 0 || index >= len(l.items) {
		l.mu.RUnlock()
		return
	}
	item := l.items[index]
	l.mu.RUnlock()
	l.updateState(func(state *uiState) {
		state.actionMenu = true
		state.actionTarget = item
		state.selected = index
		state.focusSearch = false
	})
	l.actionFocus = true
}

func (l *launcher) closeResultActions() {
	l.updateState(func(state *uiState) {
		state.actionMenu = false
		state.focusSearch = true
	})
}

func (l *launcher) updateResultActions(gtx layout.Context, index int) {
	for l.resultMore[index].Clicked(gtx) {
		l.openResultActions(index)
	}
	for {
		e, ok := gtx.Event(pointer.Filter{Target: &l.resultContext[index], Kinds: pointer.Press})
		if !ok {
			break
		}
		if e, ok := e.(pointer.Event); ok && e.Buttons&pointer.ButtonSecondary != 0 {
			l.openResultActions(index)
		}
	}
}

func (l *launcher) resultMoreButton(gtx layout.Context, index int) layout.Dimensions {
	return l.resultMore[index].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			style := material.Label(l.theme, unit.Sp(20), "…")
			style.Color = l.palette.secondary
			return style.Layout(gtx)
		})
	})
}

func (l *launcher) executeResultAction(gtx layout.Context, item g.Resource, action resultAction) {
	allowed := false
	for _, choice := range actionsForResult(item) {
		if choice.action == action {
			allowed = true
			break
		}
	}
	if !allowed {
		return
	}
	l.closeResultActions()
	switch action {
	case resultOpen:
		l.openResource(gtx, item)
	case resultReveal:
		if err := utils.RevealInFolder(item.Filepath); err != nil {
			l.message("Could not reveal the selected item in Explorer.")
		}
	case resultCopy:
		value := item.Filepath
		if item.Computed {
			value = item.Name
		}
		l.copyText(gtx, value)
	case resultAdmin:
		l.runResourceElevated(item)
	case resultHide:
		apps.BlockApplication(item)
		l.query(l.editor.Text())
	}
}

func (l *launcher) resultActionMenu(gtx layout.Context) layout.Dimensions {
	state := l.snapshot()
	choices := actionsForResult(state.actionTarget)
	for index, choice := range choices {
		for l.actionButtons[index].Clicked(gtx) {
			l.executeResultAction(gtx, state.actionTarget, choice.action)
		}
	}
	for l.actionDismiss.Clicked(gtx) {
		l.closeResultActions()
	}
	if !l.snapshot().actionMenu {
		return layout.Dimensions{}
	}
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return l.actionDismiss.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return l.pageSurface(gtx, color.NRGBA{A: 50}, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} })
			})
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			width := min(gtx.Constraints.Max.X-gtx.Dp(20), gtx.Dp(300))
			content := gtx
			content.Constraints = layout.Constraints{Min: image.Pt(width, 0), Max: image.Pt(width, gtx.Constraints.Max.Y)}
			recording := op.Record(gtx.Ops)
			dimensions := l.pageSurface(content, l.palette.surface, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					children := []layout.FlexChild{layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return l.resultText(gtx, state.actionTarget.Name, unit.Sp(12), l.palette.secondary)
						})
					})}
					for index, choice := range choices {
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return l.actionButtons[index].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								background := l.palette.surface
								if l.actionButtons[index].Hovered() || gtx.Focused(&l.actionButtons[index]) {
									background = l.palette.hover
								}
								return l.pageSurface(gtx, background, func(gtx layout.Context) layout.Dimensions {
									return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
											layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return l.pageTitle(gtx, choice.label, unit.Sp(12)) }),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.pageDescription(gtx, choice.shortcut) }),
										)
									})
								})
							})
						}))
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
				})
			})
			call := recording.Stop()
			position := popupPosition(l.pointerPosition, dimensions.Size, gtx.Constraints.Max, gtx.Dp(10))
			offset := op.Offset(position).Push(gtx.Ops)
			call.Add(gtx.Ops)
			offset.Pop()
			if l.actionFocus {
				gtx.Execute(key.FocusCmd{Tag: &l.actionButtons[0]})
				l.actionFocus = false
			}
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
	)
}

func popupPosition(position, size, bounds image.Point, margin int) image.Point {
	position.X = min(max(margin, position.X), max(margin, bounds.X-size.X-margin))
	position.Y = min(max(margin, position.Y), max(margin, bounds.Y-size.Y-margin))
	return position
}
