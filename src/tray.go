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
	systray.SetTooltip("A fast Windows navigation tool")

	mToggle := systray.AddMenuItem("Show", "Show window")
	mQuit := systray.AddMenuItem("Exit", "Exit program")

	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				showWindow()
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
