package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"winfastnav/internal/apps"
	"winfastnav/internal/documents"
	"winfastnav/internal/globals"
	"winfastnav/internal/hotkey"
	"winfastnav/internal/instance"
	"winfastnav/internal/recent"
	"winfastnav/internal/settings"
	"winfastnav/internal/utils"
	"winfastnav/internal/windowcontrol"
	"winfastnav/ui"
)

var keyboardHotkey *hotkey.Listener
var singleInstance *instance.Guard

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

	guard, acquired, err := instance.Acquire(`Local\WinFastNav`)
	if err != nil {
		log.Printf("failed to acquire single-instance guard: %v", err)
		return
	}
	if !acquired {
		if shown, showErr := windowcontrol.ShowExistingAndFocus(globals.AppName); showErr != nil {
			log.Printf("failed to focus the existing launcher: %v", showErr)
		} else if !shown {
			log.Printf("existing launcher window was not found")
		}
		return
	}
	singleInstance = guard

	settings.SetupSettings()
	recent.Load()
	apps.SetAliases(globals.AliasString)
	apps.LoadCatalog()
	utils.LoadCurrencyRates()
	ui.SetupUI()
	apps.SetCatalogChangedHandler(ui.RefreshResults)
	documents.SetChangedHandler(ui.RefreshResults)
	go documents.SetupDocs()
	go apps.MonitorCatalog()
	go func() {
		if utils.RefreshCurrencyRates() {
			ui.RefreshResults()
		}
	}()
	go listenHotkeys()
	ui.Run()
	setupTray()
}

func listenHotkeys() {
	listener, err := hotkey.Start(func() {
		ui.ToggleWindow()
	})
	if err != nil {
		log.Printf("failed to register Alt+Space hotkey: %v", err)
		return
	}
	keyboardHotkey = listener
}

func onExit() {
	if keyboardHotkey != nil {
		keyboardHotkey.Stop()
	}
	if singleInstance != nil {
		singleInstance.Release()
	}
}
