package main

import (
	"fyne.io/fyne/v2"
	"github.com/getlantern/systray"
	"os"
	g "winfastnav/internal/globals"
	"winfastnav/ui"
)

func setupTray() {
	go systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(g.IconBytes)
	systray.SetTitle(g.AppName)
	systray.SetTooltip("winfastnav: fast windows navigation")

	mToggle := systray.AddMenuItem("Show", "Show window")
	mAbout := systray.AddMenuItem("About", "Show window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Exit program")

	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				ui.ShowWindow()
			case <-mAbout.ClickedCh:
				ui.ShowWindow()
				ui.ShowAbout()
			case <-mQuit.ClickedCh:
				fyne.Do(func() {
					g.NavApp.Quit()
				})
				systray.Quit()
				os.Exit(0)
			}
		}
	}()
}
