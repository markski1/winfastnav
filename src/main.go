package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"winfastnav/internal/apps"
	"winfastnav/internal/documents"
	"winfastnav/internal/hotkey"
	"winfastnav/internal/settings"
	"winfastnav/ui"
)

var keyboardHotkey *hotkey.Listener

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
	go documents.SetupDocs()
	go apps.SetupApps()
	go listenHotkeys()
	ui.Run()
	setupTray()
}

func listenHotkeys() {
	listener, err := hotkey.Start(func() {
		ui.ShowWindow()
	})
	if err != nil {
		log.Printf("failed to register Alt+O hotkey: %v", err)
		return
	}
	keyboardHotkey = listener
}

func onExit() {
	if keyboardHotkey != nil {
		keyboardHotkey.Stop()
	}
}
