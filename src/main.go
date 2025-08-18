package main

import (
	"log"
	"os"
	"runtime/debug"
	"winfastnav/internal/apps"
	"winfastnav/internal/documents"
	g "winfastnav/internal/globals"
	"winfastnav/internal/settings"
	"winfastnav/ui"

	"github.com/robotn/gohook"
)

var (
	keyboardHook chan hook.Event
)

func main() {
	// Setup file log for panics to try and hunt down a crash when resuming from sleep.
	defer func() {
		if r := recover(); r != nil {
			_ = os.MkdirAll("logs", 0o755)
			f, _ := os.Create("logs/panic.log")
			_ = f.Close()
			log.Printf("panic: %v\n%s", r, debug.Stack())
			if f != nil {
				_, _ = f.Write(debug.Stack())
			}
		}
	}()

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

	keyboardHook = hook.Start(75)
	<-hook.Process(keyboardHook)
}

func onExit() {
	hook.End()
	if keyboardHook != nil {
		close(keyboardHook)
	}
}
