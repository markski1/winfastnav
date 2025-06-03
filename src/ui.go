package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/getlantern/systray"
	"log"
	"os"
	"runtime/debug"
	d "winfastnav/assets"
	w "winfastnav/widgets"
)

var (
	shown bool = false
)

func setupUI() {
	log.Printf("Preparing UI")

	// Attempt to create a borderless window as a 'splash'.
	if drv, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
		w.NavWindow = drv.CreateSplashWindow()
		w.NavWindow.SetTitle("winfastnav")
	} else {
		w.NavWindow = w.NavApp.NewWindow("winfastnav")
	}

	w.NavWindow.Resize(fyne.NewSize(400, 250))
	w.NavWindow.SetFixedSize(true)
	w.NavWindow.CenterOnScreen()
	resourceIcon := fyne.NewStaticResource("icon.ico", iconBytes)
	w.NavWindow.SetIcon(resourceIcon)

	w.InputEntry = w.NewCustomEntry(func() {
		fyne.Do(func() {
			if len(w.InputEntry.Text) > 0 {
				w.NavWindow.Canvas().Focus(w.ResultList)
			}
		})
	})
	w.InputEntry.SetPlaceHolder("Start typing, ESC to hide")

	w.InputEntry.OnChanged = func(s string) {
		updateResultList(s)
	}

	w.ResultList = w.NewCustomList([]d.App{}, func(idx int, app d.App) {
		openProgram(app.ExecPath)
	})

	updateContent()
	showWindow()

	// Don't close on X, hide instead.
	w.NavWindow.SetCloseIntercept(func() {
		hideWindow()
	})
	log.Printf("Done")
}

func updateContent() {
	bottomHBox := container.NewCenter(
		container.NewHBox(
			widget.NewButton("Help", func() {
				showHelp()
			}),
			widget.NewButton("Settings", func() {
				showSettings()
			}),
			widget.NewButton("Quit", func() {
				fyne.Do(func() {
					w.NavApp.Quit()
				})
				systray.Quit()
				os.Exit(0)
			}),
		),
	)

	bottomHBox.Resize(fyne.NewSize(400, 20))

	content := container.NewPadded(
		container.NewBorder(
			w.InputEntry,
			bottomHBox,
			nil, nil,
			w.ResultList,
		),
	)

	w.NavWindow.SetContent(content)
}

func showSettings() {
	searchStringEntry := widget.NewEntry()
	searchStringEntry.SetText(d.SearchString)
	searchStringEntry.OnChanged = func(s string) {
		d.SearchString = s
		_ = d.SetSetting("searchstring", s)
	}

	searchStringBox := container.NewVBox(
		widget.NewLabel("Search string"),
		searchStringEntry,
	)

	searchStringBox.Resize(fyne.NewSize(400, 20))

	blocklistBox := container.NewHBox(
		widget.NewLabel(fmt.Sprintf("Blocklist (%d)", len(d.ExecBlocklist))),
		widget.NewButton("Clear Blocklist", func() {
			d.UnblockAllApplications()
			dialog.NewInformation("Blocklist cleared", "All apps have been unblocked", w.NavWindow).Show()
		}),
	)

	topVBox := container.NewVBox(
		searchStringBox,
		widget.NewSeparator(),
		blocklistBox,
	)

	bottomVBox := container.NewVBox(
		widget.NewButton("OK", func() {
			updateContent()
		}),
	)

	content := container.NewPadded(
		container.NewBorder(
			topVBox,
			bottomVBox,
			nil,
			nil,
		),
	)

	w.NavWindow.SetContent(content)
}

func showHelp() {
	topVBox := container.NewVBox(
		widget.NewLabel(
			"Shortcuts:\n" +
				"ALT + O: Summon\n" +
				"ESC: Hide\n" +
				"Delete: Hide app",
		),
	)

	midVBox := container.NewVBox(
		widget.NewLabel(
			"Prefixes:\n" +
				"@: Internet search\n",
		),
	)

	bottomVBox := container.NewVBox(
		widget.NewButton("OK", func() {
			updateContent()
		}),
	)

	content := container.NewPadded(
		container.NewBorder(
			topVBox,
			bottomVBox,
			nil,
			nil,
			midVBox,
		),
	)

	w.NavWindow.SetContent(content)
}

func showAbout() {
	topVBox := container.NewVBox(
		widget.NewLabel("winfastnav: fast windows navigation"),
	)

	bottomVBox := container.NewVBox(
		widget.NewLabel("markski.ar\ngithub.com/markski1"),
		widget.NewButton("OK", func() {
			updateContent()
		}),
	)

	content := container.NewPadded(
		container.NewBorder(
			topVBox,
			bottomVBox,
			nil,
			nil,
		),
	)

	w.NavWindow.SetContent(content)
}

func updateResultList(needle string) {
	if len(needle) == 0 {
		setResultListFor(nil)
	} else {
		apps := findAppResults(needle)
		setResultListFor(apps)
	}

	updateContent()
}

func setResultListFor(appList []d.App) {
	if appList == nil {
		appList = []d.App{}
	}

	w.ResultList.UpdateItems(appList)
}

func showWindow() {
	shown = true
	fyne.Do(func() {
		w.NavWindow.Show()
		w.InputEntry.SetText("")
		w.NavWindow.RequestFocus()
		w.NavWindow.Canvas().Focus(w.InputEntry)
	})
}

func hideWindow() {
	shown = false
	fyne.Do(func() {
		updateResultList("")
		w.NavWindow.Hide()
	})
	debug.FreeOSMemory()
}
