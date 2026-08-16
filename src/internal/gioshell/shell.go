package gioshell

import (
	"fmt"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"winfastnav/internal/presentation"
)

type Shell struct {
	controller *presentation.Controller
	window     app.Window
	ops        op.Ops
	theme      *material.Theme
}

func New(controller *presentation.Controller, title string) *Shell {
	shell := &Shell{
		controller: controller,
		theme:      material.NewTheme(),
	}
	shell.window.Option(
		app.Title(title),
		app.Size(unit.Dp(425), unit.Dp(300)),
		app.MinSize(unit.Dp(425), unit.Dp(300)),
	)
	controller.SetInvalidator(shell.window.Invalidate)
	return shell
}

func (s *Shell) Run() error {
	defer s.controller.SetInvalidator(nil)

	for {
		switch event := s.window.Event().(type) {
		case app.DestroyEvent:
			return event.Err
		case app.FrameEvent:
			s.ops.Reset()
			gtx := app.NewContext(&s.ops, event)
			s.layout(gtx)
			event.Frame(&s.ops)
		}
	}
}

func (s *Shell) layout(gtx layout.Context) layout.Dimensions {
	state := s.controller.Snapshot()
	return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.H6(s.theme, "winfastnav").Layout),
			layout.Rigid(material.Body1(s.theme, pageLabel(state.Page)).Layout),
			layout.Rigid(material.Body1(s.theme, fmt.Sprintf("Mode: %d", state.Mode)).Layout),
			layout.Rigid(material.Body1(s.theme, fmt.Sprintf("Query: %s", state.Query)).Layout),
			layout.Rigid(material.Body1(s.theme, state.Message).Layout),
		)
	})
}

func pageLabel(page presentation.Page) string {
	switch page {
	case presentation.PageMenu:
		return "Menu"
	case presentation.PageHelp:
		return "Help"
	case presentation.PageSettings:
		return "Settings"
	case presentation.PageAbout:
		return "About"
	default:
		return "Launcher"
	}
}
