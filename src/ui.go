package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func setupUI() {
	navWindow.Resize(fyne.NewSize(400, 300))
	navWindow.SetFixedSize(true)
	navWindow.Hide()

	inputEntry = widget.NewEntry()
	inputEntry.SetPlaceHolder("Enter navigation...")

	inputEntry.OnSubmitted = func(text string) {
		navWindow.Hide()
	}

	hideButton := widget.NewButton("Hide", func() {
		navWindow.Hide()
	})

	instructionLabel := widget.NewLabel("Alt+O to summon, ESC to hide.")
	titleLabel := widget.NewLabel("winfastnav")
	titleLabel.TextStyle.Bold = true

	content := container.NewVBox(
		titleLabel,
		inputEntry,
		widget.NewSeparator(),
		hideButton,
		instructionLabel,
	)

	navWindow.SetContent(content)

	// Don't close, hide
	navWindow.SetCloseIntercept(func() {
		navWindow.Hide()
	})
}

func showWindow() {
	fyne.Do(func() {
		navWindow.Show()
		navWindow.RequestFocus()
		inputEntry.FocusGained()
	})
}

func hideWindow() {
	fyne.Do(func() {
		navWindow.Hide()
	})
}
