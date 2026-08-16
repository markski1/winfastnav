package main

import (
	"fmt"
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
			if err := os.MkdirAll(dir, 0o700); err != nil {
				log.Printf("failed to create panic log directory: %v", err)
				return
			}
			f, _ := os.Create(filepath.Join(dir, "panic.log"))
			if f != nil {
				_, _ = fmt.Fprintf(f, "panic: %v\n", r)
				_, _ = f.Write(debug.Stack())
				_ = f.Close()
			}
			log.Printf("panic: %v\n%s", r, debug.Stack())
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
}
