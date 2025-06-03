package ui

import (
	_ "embed"
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
	"winfastnav/internal/apps"
	g "winfastnav/internal/globals"
	"winfastnav/internal/settings"
	w "winfastnav/ui/widgets"
)

var (
	InputEntry *w.CustomEntry
	ResultList *w.CustomList
)

func SetupUI() {
	log.Printf("Preparing UI")

	// Attempt to create a borderless window as a 'splash'.
	if drv, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
		g.NavWindow = drv.CreateSplashWindow()
		g.NavWindow.SetTitle("winfastnav")
	} else {
		g.NavWindow = g.NavApp.NewWindow("winfastnav")
	}

	g.NavWindow.Resize(fyne.NewSize(400, 250))
	g.NavWindow.SetFixedSize(true)
	g.NavWindow.CenterOnScreen()
	resourceIcon := fyne.NewStaticResource("icon.ico", g.IconBytes)
	g.NavWindow.SetIcon(resourceIcon)

	InputEntry = w.NewCustomEntry(func() {
		fyne.Do(func() {
			if len(InputEntry.Text) > 0 {
				g.NavWindow.Canvas().Focus(ResultList)
			}
		})
	})
	InputEntry.SetPlaceHolder("Start typing, ESC to hide")
	InputEntry.OnSubmitted = func(s string) {
		g.NavWindow.Canvas().Focus(ResultList)
	}

	InputEntry.OnChanged = func(s string) {
		updateResultList(s)
	}

	ResultList = w.NewCustomList([]g.App{}, InputEntry, func(idx int, app g.App) {
		apps.OpenProgram(app.ExecPath)
		HideWindow()
	})

	updateContent()
	ShowWindow()

	// Don't close on X, hide insteag.
	g.NavWindow.SetCloseIntercept(func() {
		HideWindow()
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
					g.NavApp.Quit()
				})
				systray.Quit()
				os.Exit(0)
			}),
		),
	)

	bottomHBox.Resize(fyne.NewSize(400, 20))

	content := container.NewPadded(
		container.NewBorder(
			InputEntry,
			bottomHBox,
			nil, nil,
			ResultList,
		),
	)

	g.NavWindow.SetContent(content)
}

func showSettings() {
	searchStringEntry := widget.NewEntry()
	searchStringEntry.SetText(g.SearchString)
	searchStringEntry.OnChanged = func(s string) {
		g.SearchString = s
		_ = settings.SetSetting("searchstring", s)
	}

	searchStringBox := container.NewVBox(
		widget.NewLabel("Search string"),
		searchStringEntry,
	)

	searchStringBox.Resize(fyne.NewSize(400, 20))

	blocklistBox := container.NewHBox(
		widget.NewLabel(fmt.Sprintf("Blocklist (%d)", len(g.ExecBlocklist))),
		widget.NewButton("Clear Blocklist", func() {
			apps.UnblockAllApplications()
			dialog.NewInformation("Blocklist cleared", "All apps have been unblocked", g.NavWindow).Show()
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

	g.NavWindow.SetContent(content)
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

	g.NavWindow.SetContent(content)
}

func ShowAbout() {
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

	g.NavWindow.SetContent(content)
}

func updateResultList(needle string) {
	if len(needle) == 0 {
		setResultListFor(nil)
	} else {
		getapps := apps.FindAppResults(needle)
		setResultListFor(getapps)
	}

	updateContent()
}

func setResultListFor(appList []g.App) {
	if appList == nil {
		appList = []g.App{}
	}

	ResultList.UpdateItems(appList)
}

func ShowWindow() {
	g.Shown = true
	fyne.Do(func() {
		g.NavWindow.Show()
		InputEntry.SetText("")
		g.NavWindow.RequestFocus()
		g.NavWindow.Canvas().Focus(InputEntry)
	})
}

func HideWindow() {
	g.Shown = false
	fyne.Do(func() {
		updateResultList("")
		g.NavWindow.Hide()
	})
	debug.FreeOSMemory()
}
