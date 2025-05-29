package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	instructionLabel *widget.Label = widget.NewLabel("ESC to hide.")
	resultList       *widget.List
	shown            bool = false
)

func setupUI() {
	navWindow.Resize(fyne.NewSize(450, 275))
	navWindow.SetFixedSize(true)
	navWindow.Hide()

	inputEntry = widget.NewEntry()
	inputEntry.SetPlaceHolder("Start typing, ESC to hide.")

	inputEntry.OnChanged = func(s string) {
		updateResultList(s)
	}

	widget.NewSeparator()

	// Setup resultList empty
	resultList = makeResultsList(nil)

	updateContent()

	showWindow()

	// Don't close, hide
	navWindow.SetCloseIntercept(func() {
		navWindow.Hide()
	})
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
		resultList = makeResultsList(nil)
	} else {
		apps := findAppResults(needle)
		keys := make([]string, 0, len(apps))
		for _, key := range apps {
			keys = append(keys, key.Name)
		}

		resultList = makeResultsList(keys)
	}

	updateContent()
}

func makeResultsList(keys []string) *widget.List {
	if keys == nil {
		keys = []string{}
	}

	newList := widget.NewList(
		func() int {
			return len(keys)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			label.SetText(keys[i])
		},
	)

	return newList
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
