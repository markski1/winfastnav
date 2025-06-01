package main

import (
	_ "embed"
	"fyne.io/fyne/v2"
	"github.com/getlantern/systray"
	"log"
	"os"
	"time"
	w "winfastnav/widgets"
)

var (
	//go:embed assets/icon.ico
	iconBytes []byte
)

func setupTray() {
	log.Printf("Preparing tray")
	go systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(iconBytes)
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
				fyne.Do(func() {
					w.NavApp.Quit()
				})
				systray.Quit()
				os.Exit(0)
			}
			time.Sleep(1000 * time.Millisecond)
		}
	}()
	log.Printf("Done")
}
