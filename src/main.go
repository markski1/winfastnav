package main

import (
	"github.com/robotn/gohook"
	"winfastnav/internal/apps"
	"winfastnav/internal/documents"
	g "winfastnav/internal/globals"
	"winfastnav/internal/settings"
	"winfastnav/ui"
)

var (
	keyboardHook chan hook.Event
)

func main() {
	settings.SetupSettings()
	ui.SetupUI()
	setupTray()
	go documents.SetupDocs()
	go apps.SetupApps()
	go listenHotkeys()
	g.NavApp.Run()
}

func listenHotkeys() {
	hook.Register(hook.KeyDown, []string{"alt", "o"}, func(e hook.Event) {
		if !g.Shown {
			ui.ShowWindow()
		}
	})

	// Register escape key to hide window when it's focused
	hook.Register(hook.KeyDown, []string{"esc"}, func(e hook.Event) {
		if g.Shown {
			if !g.ShowingMain {
				ui.ShowWindow()
			} else {
				ui.HideWindow()
			}
		}
	})

	keyboardHook = hook.Start()
	defer hook.End()
	<-hook.Process(keyboardHook)
}

func onExit() {
	if keyboardHook != nil {
		close(keyboardHook)
	}
}
