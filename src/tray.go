package main

import (
	"github.com/getlantern/systray"
	"os"
	g "winfastnav/internal/globals"
	"winfastnav/ui"
)

func setupTray() {
	systray.Run(onReady, onExit)
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
				go ui.ShowWindow()
			case <-mAbout.ClickedCh:
				go func() {
					ui.ShowWindow()
					ui.ShowAbout()
				}()
			case <-mQuit.ClickedCh:
				ui.Quit()
				systray.Quit()
				os.Exit(0)
			}
		}
	}()
}
