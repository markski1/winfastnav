package main

import (
	"log"
	"os"
	"path/filepath"
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
			appData := os.Getenv("APPDATA")
			dir := filepath.Join(appData, "winfastnav")
			f, _ := os.Create(filepath.Join(dir, "panic.log"))
			log.Printf("panic: %v\n%s", r, debug.Stack())
			if f != nil {
				_, _ = f.Write(debug.Stack())
			}
			_ = f.Close()
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
