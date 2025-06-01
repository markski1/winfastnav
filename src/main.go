/*
	Outstanding issues:

	- Figure out best way to make window frameless.
	- Find open windows
*/

package main

import (
	"github.com/robotn/gohook"
	"log"
	ui "winfastnav/widgets"
)

var (
	keyboardHook chan hook.Event
)

func main() {
	setupUI()
	setupTray()
	setupApps()
	go listenHotkeys()
	log.Printf("BEGIN")
	ui.NavApp.Run()
}

func listenHotkeys() {
	log.Printf("Preparing hotkey listeners")
	hook.Register(hook.KeyDown, []string{"alt", "o"}, func(e hook.Event) {
		showWindow()
	})

	// Register escape key to hide window when it's focused
	hook.Register(hook.KeyDown, []string{"esc"}, func(e hook.Event) {
		if shown {
			hideWindow()
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
