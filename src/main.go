/*
	Outstanding issues:

	- Still need an icon and not the example .ico file I found somewhere.
	- Figure out best way to make window frameless.
	- How to best divide this project?
	- Keyboard selection and opening programs
	- Find focused windows
*/

package main

import (
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
	"github.com/robotn/gohook"
)

var (
	navApp       = app.New()
	navWindow    = navApp.NewWindow("winfastnav")
	inputEntry   *widget.Entry
	keyboardHook chan hook.Event
)

func main() {
	setupUI()
	setupTray()
	setupApps()
	go listenHotkeys()
	navApp.Run()
}

func listenHotkeys() {
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
}

func onExit() {
	if keyboardHook != nil {
		close(keyboardHook)
	}
}
