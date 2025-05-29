package main

import (
	_ "embed"
	"github.com/getlantern/systray"
	"os"
)

var (
	//go:embed assets/icon.ico
	iconBytes []byte
)

func setupTray() {
	go systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(getIcon())
	systray.SetTitle("winfastnav")
	systray.SetTooltip("winfastnav: fast windows navigation")

	mToggle := systray.AddMenuItem("Show", "Show window")
	mAbout := systray.AddMenuItem("About", "Show window")
	mQuit := systray.AddMenuItem("Exit", "Exit program")

	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				showWindow()
			case <-mAbout.ClickedCh:
				showWindow()
				showAbout()
			case <-mQuit.ClickedCh:
				systray.Quit()
				navApp.Quit()
				os.Exit(0)
			}
		}
	}()
}

func getIcon() []byte {
	return iconBytes
}
