package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"log"
	"runtime"
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

	w.NavWindow.Resize(fyne.NewSize(400, 225))
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
	content := container.NewBorder(
		w.InputEntry,
		nil, nil, nil,
		w.ResultList,
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

	content := container.NewBorder(
		topVBox,
		bottomVBox,
		nil,
		nil,
	)

	fyne.Do(func() {
		w.NavWindow.SetContent(content)
	})
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
	runtime.GC()
	debug.FreeOSMemory()
}
