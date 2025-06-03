package main

import (
	"github.com/robotn/gohook"
	"log"
	"winfastnav/internal/apps"
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
	apps.SetupApps()
	go listenHotkeys()
	log.Printf("BEGIN")
	g.NavApp.Run()
}

func listenHotkeys() {
	log.Printf("Preparing hotkey listeners")
	hook.Register(hook.KeyDown, []string{"alt", "o"}, func(e hook.Event) {
		ui.ShowWindow()
	})

	// Register escape key to hide window when it's focused
	hook.Register(hook.KeyDown, []string{"esc"}, func(e hook.Event) {
		if g.Shown {
			ui.HideWindow()
		}
	})

	keyboardHook = hook.Start()
	defer hook.End()
	<-hook.Process(keyboardHook)
	log.Printf("Done")
}

func onExit() {
	if keyboardHook != nil {
		close(keyboardHook)
	}
}
