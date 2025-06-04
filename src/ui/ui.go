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
	"strconv"
	"strings"
	"winfastnav/internal/apps"
	g "winfastnav/internal/globals"
	"winfastnav/internal/settings"
	"winfastnav/internal/utils"
	w "winfastnav/ui/widgets"
)

var (
	InputEntry      *w.CustomEntry
	ResultList      *w.CustomList[g.App]
	inputContainer  *fyne.Container
	openSize        = fyne.NewSize(425, 300)
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

	g.NavWindow.Resize(openSize)
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

	ResultList = w.NewCustomList[]([]g.App{}, InputEntry, func(app g.App) string { return app.Name }, func(idx int, app g.App) {
		apps.OpenProgram(app.ExecPath)
		HideWindow()
	})

	updateContent(nil)
	ShowWindow()

	// Don't close on X, hide insteag.
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
		g.SearchString = s
		_ = settings.SetSetting("searchstring", s)
	}

	searchStringBox := container.NewVBox(
		widget.NewLabel("Search string"),
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
				"@: Internet search\n",
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

func updateResultList(inputText string) {
	// only if not in app choice mode
	if choosingOpenApp {
		return
	}
	if len(inputText) > 0 {
		if inputText[0] == '@' {
			updateContent(widget.NewLabel("Internet search: " + inputText[1:]))
			return
		}

		// If it's a math op, eval and show result.
		if utils.IsMath(inputText) {
			// remove spaces and eval
			inputText := strings.ReplaceAll(inputText, " ", "")
			result, err := utils.EvalMath(inputText)
			// If we cannot eval then assume IsMath false positive and proceed w/ results.
			if err == nil {
				updateContent(container.NewVBox(
					widget.NewLabel(result),
				))
				return
			}
		}

		getapps := apps.FindAppResults(inputText)
		setResultListFor(getapps)
	} else {
		setResultListFor(nil)
	}

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
		if choosingOpenApp {
			num, err := strconv.Atoi(inputText)
			if err == nil {
				HideWindow()
				apps.FocusWindow(num)
				return
			}
		}
	}
	// Otherwise attempt to focus the list.
	g.NavWindow.Canvas().Focus(ResultList)
}

func setResultListFor(appList []g.App) {
	if appList == nil {
		appList = []g.App{}
	}

	ResultList.UpdateItems(appList)
}

func SetChooseOpenApps() {
	choosingOpenApp = true
	openAppList := apps.GetOpenWindows()
	updateContent(container.NewVBox(
		widget.NewLabel("Input option number:"),
		widget.NewLabel(openAppList),
	))
}

func ShowWindow() {
	g.Shown = true
	choosingOpenApp = false
	fyne.Do(func() {
		g.NavWindow.Show()
		InputEntry.SetText("")
		g.NavWindow.RequestFocus()
		g.NavWindow.Canvas().Focus(InputEntry)
		updateContent(nil)
	})
}

func HideWindow() {
	g.Shown = false
	fyne.Do(func() {
		g.NavWindow.Hide()
	})
	debug.FreeOSMemory()
}
