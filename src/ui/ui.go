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
	"strconv"
	"strings"
	"time"
	"winfastnav/internal/apps"
	"winfastnav/internal/core"
	g "winfastnav/internal/globals"
	"winfastnav/internal/utils"
	w "winfastnav/ui/widgets"
)

var (
	InputEntry      *w.CustomEntry
	ResultList      *w.CustomList[g.App]
	openProgramList *w.CustomList[string]
	inputContainer  *fyne.Container
	choosingOpenApp = false
)

func SetupUI() {
	log.Printf("Preparing UI")

	g.NavApp.Settings().SetTheme(&wfnTheme{})

	// Attempt to create a borderless window as a 'splash'.
	if drv, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
		g.NavWindow = drv.CreateSplashWindow()
		g.NavWindow.SetTitle(g.AppName)
	} else {
		g.NavWindow = g.NavApp.NewWindow(g.AppName)
	}

	g.NavWindow.Resize(fyne.NewSize(425, 300))
	g.NavWindow.SetFixedSize(true)
	g.NavWindow.CenterOnScreen()
	resourceIcon := fyne.NewStaticResource("icon.ico", g.IconBytes)
	g.NavWindow.SetIcon(resourceIcon)

	InputEntry = w.NewCustomEntry(func() {
		fyne.Do(func() {
			if len(InputEntry.Text) > 0 {
				if !choosingOpenApp {
					g.NavWindow.Canvas().Focus(ResultList)
				}
			}
			if choosingOpenApp {
				g.NavWindow.Canvas().Focus(openProgramList)
			}
		})
	})
	InputEntry.OnSubmitted = func(s string) {
		updateSubmitContent(s)
	}

	InputEntry.OnChanged = func(s string) {
		updateResultList(s)
	}

	inputContainer = container.NewBorder(
		nil, nil, nil,
		widget.NewButton("Menu", func() {
			showMenu()
		}),
		InputEntry,
	)

	ResultList = w.NewCustomList([]g.App{}, InputEntry, func(app g.App) string { return app.Name }, func(idx int, app g.App) {
		apps.OpenProgram(app.ExecPath)
		HideWindow()
	})

	openProgramList = w.NewCustomList([]string{}, InputEntry, func(s string) string { return s }, func(idx int, s string) {
		apps.FocusWindow(idx + 1)
		HideWindow()
	})

	updateContent(nil)
	ShowWindow()

	// Don't close on X, hide instead.
	g.NavWindow.SetCloseIntercept(func() {
		HideWindow()
	})
	log.Printf("Done")
}

func updateContent(aContent fyne.CanvasObject) {
	fyne.Do(func() {
		g.NavWindow.SetContent(container.NewPadded(
			container.NewBorder(
				inputContainer,
				nil, nil, nil,
				aContent,
			),
		))
		g.NavWindow.RequestFocus()
		g.NavWindow.Canvas().Focus(InputEntry)
	})
}

func showContent(aContent fyne.CanvasObject) {
	bottomVBox := container.NewVBox(
		widget.NewButton("OK", func() {
			updateContent(nil)
		}),
	)

	content := container.NewPadded(
		container.NewBorder(
			aContent,
			bottomVBox,
			nil,
			nil,
		),
	)

	g.NavWindow.SetContent(content)
}

func showMenu() {
	content := container.NewVBox(
		widget.NewButton("Help", func() {
			showHelp()
		}),
		widget.NewButton("Settings", func() {
			showSettings()
		}),
		widget.NewButton("About", func() {
			ShowAbout()
		}),
		widget.NewButton("Quit", func() {
			fyne.Do(func() {
				g.NavApp.Quit()
			})
			systray.Quit()
			os.Exit(0)
		}),
	)
	showContent(content)
}

func showSettings() {
	searchStringEntry := widget.NewEntry()
	searchStringEntry.SetText(g.SearchString)
	searchStringEntry.OnChanged = func(s string) {
		core.UpdateSearchSetting(s)
	}

	searchStringBox := container.NewVBox(
		widget.NewLabel("HandleTextInput string"),
		searchStringEntry,
	)

	blocklistBox := container.NewHBox(
		widget.NewLabel(fmt.Sprintf("Blocklist (%d)", len(g.ExecBlocklist))),
		widget.NewButton("Clear Blocklist", func() {
			apps.UnblockAllApplications()
			dialog.NewInformation("Blocklist cleared", "All apps have been unblocked", g.NavWindow).Show()
		}),
	)

	content := container.NewVBox(
		searchStringBox,
		widget.NewSeparator(),
		blocklistBox,
	)

	showContent(content)
}

func showHelp() {
	first := container.NewVBox(
		widget.NewLabel(
			"Shortcuts:\n" +
				"ALT + O: Summon\n" +
				"ESC: Hide\n" +
				"Delete: Hide app",
		),
	)

	second := container.NewVBox(
		widget.NewLabel(
			"Prefixes:\n"+
				"@: Internet search\n"+
				"!: GPT prompt\n",
		),
		widget.NewLabel(
			"Math:\n"+
				"Supported: + - * /\n"+
				"Just write an operation and see the result.",
		),
	)

	content := container.NewVBox(first, second)

	showContent(content)
}

func ShowAbout() {
	topVBox := container.NewVBox(
		widget.NewLabel("winfastnav: fast windows navigation"),
	)

	bottomVBox := container.NewVBox(
		widget.NewLabel("markski.ar\ngithub.com/markski1"),
		widget.NewButton("OK", func() {
			updateContent(nil)
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

func updateResultList(input string) {
	if choosingOpenApp {
		return
	}
	getApps, mathResult := core.HandleTextInput(input)
	if mathResult != nil {
		updateContent(widget.NewLabel(*mathResult))
		return
	}
	ResultList.UpdateItems(getApps)
	updateContent(ResultList)
}

func updateSubmitContent(inputText string) {
	if len(inputText) > 0 {
		if inputText[0] == '@' {
			return
		}
		// If it's a math op, set the result as the new input text
		if utils.IsMath(inputText) {
			inputText := strings.ReplaceAll(inputText, " ", "")
			result, err := utils.EvalMath(inputText)
			if err == nil {
				InputEntry.SetText(result)
				return
			}
		}
		if utils.StartsWith(inputText, "!") {
			updateContent(widget.NewLabel("Please wait..."))
			prompt := inputText[1:]
			go func(p string) {
				result := utils.MakeGPTReq(p)
				fyne.Do(func() {
					updateContent(widget.NewLabel(result))
				})
			}(prompt)
			return
		}
		if choosingOpenApp {
			num, err := strconv.Atoi(inputText)
			if err == nil {
				HideWindow()
				apps.FocusWindow(num)
				return
			}
		}
		g.NavWindow.Canvas().Focus(ResultList)
	}
	// Otherwise attempt to focus the list.
	if choosingOpenApp {
		g.NavWindow.Canvas().Focus(openProgramList)
		return
	}
}

func SetChooseOpenApps() {
	choosingOpenApp = true
	fyne.Do(func() {
		InputEntry.SetPlaceHolder("Choose window...")
	})
	openAppList := apps.GetOpenWindows()
	openProgramList.UpdateItems(openAppList)
	updateContent(openProgramList)
}

func ShowWindow() {
	g.Shown = true
	choosingOpenApp = false
	fyne.Do(func() {
		InputEntry.SetPlaceHolder("Program search...")
		g.NavWindow.Show()
		time.Sleep(25 * time.Millisecond)
		g.NavWindow.RequestFocus()
		g.NavWindow.Canvas().Focus(InputEntry)
	})
}

func HideWindow() {
	g.Shown = false
	fyne.Do(func() {
		g.NavWindow.Hide()
		InputEntry.SetText("")
		updateContent(nil)
	})
}
