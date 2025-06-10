package main

import (
	"github.com/robotn/gohook"
	"log"
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
	go setupTray()
	go documents.SetupDocs()
	go apps.SetupApps()
	go listenHotkeys()
	log.Printf("BEGIN")
	g.NavApp.Run()
}

func listenHotkeys() {
	log.Printf("Preparing hotkey listeners")
	hook.Register(hook.KeyDown, []string{"alt", "o"}, func(e hook.Event) {
		if !g.Shown {
			ui.ShowWindow()
		} else {
			if g.CurrentMode == g.ModeProgramSearch {
				ui.SetMode(g.ModeChoosingProgram)
			} else {
				ui.SetMode(g.ModeProgramSearch)
			}
		}
	})

	hook.Register(hook.KeyDown, []string{"alt", "d"}, func(e hook.Event) {
		if g.Shown {
			ui.SetMode(g.ModeDocumentSearch)
		}
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
