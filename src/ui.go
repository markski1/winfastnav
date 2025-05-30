package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"log"
	w "winfastnav/widgets"
)

var (
	navWindow      = navApp.NewWindow("winfastnav")
	shown     bool = false

	inputEntry *w.CustomEntry

	resultList *widget.List
	execPaths  []string
)

func setupUI() {
	log.Printf("Preparing UI")
	navWindow.Resize(fyne.NewSize(450, 275))
	navWindow.SetFixedSize(true)
	navWindow.CenterOnScreen()
	resourceIcon := fyne.NewStaticResource("icon.ico", iconBytes)
	navWindow.SetIcon(resourceIcon)

	inputEntry = w.NewCustomEntry(func() {
		fyne.Do(func() {
			if len(inputEntry.Text) > 0 {
				navWindow.Canvas().Focus(resultList)
			}
		})
	})
	inputEntry.SetPlaceHolder("Start typing, ESC to hide")

	inputEntry.OnChanged = func(s string) {
		updateResultList(s)
	}

	updateResultList("")

	showWindow()

	// Don't close on X, hide instead.
	navWindow.SetCloseIntercept(func() {
		hideWindow()
	})
	log.Printf("Done")
}

func updateContent() {
	content := container.NewBorder(
		inputEntry,
		nil, nil, nil,
		resultList,
	)

	navWindow.SetContent(content)
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
		navWindow.SetContent(content)
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

func setResultListFor(appList []App) {
	if appList == nil {
		appList = []App{}
	}

	execPaths = make([]string, len(appList))
	for i := range appList {
		execPaths[i] = appList[i].ExecPath
	}

	resultList = widget.NewList(
		func() int {
			return len(appList)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i int, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.SetText(appList[i].Name)
		},
	)

	resultList.OnSelected = func(id int) {
		openProgram(id, execPaths)
	}
}

func showWindow() {
	shown = true
	fyne.Do(func() {
		navWindow.Show()
		inputEntry.SetText("")
		navWindow.RequestFocus()
		navWindow.Canvas().Focus(inputEntry)
	})
}

func hideWindow() {
	shown = false
	fyne.Do(func() {
		updateResultList("")
		navWindow.Hide()
	})
}
