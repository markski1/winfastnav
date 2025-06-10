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
	"winfastnav/internal/documents"
	g "winfastnav/internal/globals"
	"winfastnav/internal/utils"
	w "winfastnav/ui/widgets"
)

var (
	InputEntry         *w.CustomEntry
	ProgramResultList  *w.CustomList[g.App]
	DocumentResultList *w.CustomList[g.Document]
	openProgramList    *w.CustomList[string]
	inputContainer     *fyne.Container
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
				if g.CurrentMode == g.ModeProgramSearch {
					g.NavWindow.Canvas().Focus(ProgramResultList)
				}
				if g.CurrentMode == g.ModeDocumentSearch {
					g.NavWindow.Canvas().Focus(DocumentResultList)
				}
			}
			if g.CurrentMode == g.ModeChoosingProgram {
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

	ProgramResultList = w.NewCustomList([]g.App{}, InputEntry, func(app g.App) string { return app.Name }, func(idx int, app g.App) {
		err := apps.OpenProgram(app.ExecPath)
		if err != nil {
			log.Printf("Error opening program: %v", err)
			MainShowText("Sorry, there was an error opening the program.")
			return
		}
		HideWindow()
	})

	DocumentResultList = w.NewCustomList([]g.Document{}, InputEntry, func(doc g.Document) string { return doc.Filename }, func(idx int, doc g.Document) {
		err := documents.OpenFile(doc.Path)
		if err != nil {
			log.Printf("Error opening file: %v", err)
			MainShowText("Sorry, there was an error opening the file.")
			return
		}
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
				"@: Internet search\n"+
				"!: GPT prompt\n",
		),
		widget.NewLabel(
			"Modes:\n"+
				"ALT + O: Program switch\n"+
				"ALT + D: Document search\n",
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

func MainShowText(text string) {
	formattedText := utils.WrapTextByWords(text, 64)
	updateContent(widget.NewLabel(formattedText))
}

func updateResultList(input string) {
	if g.CurrentMode == g.ModeChoosingProgram {
		return
	}
	listGet, mathResult := core.HandleTextInput(input)
	if mathResult != nil {
		MainShowText(*mathResult)
		return
	}
	if g.CurrentMode == g.ModeProgramSearch {
		appList := make([]g.App, 0, len(listGet))
		for _, item := range listGet {
			if app, ok := item.(g.App); ok {
				appList = append(appList, app)
			}
		}
		ProgramResultList.UpdateItems(appList)
		updateContent(ProgramResultList)
		return
	} else if g.CurrentMode == g.ModeDocumentSearch {
		docList := make([]g.Document, 0, len(listGet))
		for _, item := range listGet {
			if doc, ok := item.(g.Document); ok {
				docList = append(docList, doc)
			}
		}
		DocumentResultList.UpdateItems(docList)
		updateContent(DocumentResultList)
		return
	}
	updateContent(nil)
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
			MainShowText("Please wait...")
			prompt := inputText[1:]
			go func(p string) {
				result := utils.MakeGPTReq(p)
				fyne.Do(func() {
					MainShowText(result)
				})
			}(prompt)
			return
		}
		if g.CurrentMode == g.ModeChoosingProgram {
			num, err := strconv.Atoi(inputText)
			if err == nil {
				HideWindow()
				apps.FocusWindow(num)
				return
			}
		}
		g.NavWindow.Canvas().Focus(ProgramResultList)
	}
	// Otherwise attempt to focus the list.
	if g.CurrentMode == g.ModeChoosingProgram {
		g.NavWindow.Canvas().Focus(openProgramList)
		return
	}
}

func SetMode(newMode int) {
	g.CurrentMode = newMode
	if newMode == g.ModeChoosingProgram {
		fyne.Do(func() {
			InputEntry.SetPlaceHolder("Choose window...")
		})
		openAppList := apps.GetOpenWindows()
		openProgramList.UpdateItems(openAppList)
		updateContent(openProgramList)
		return
	}
	if newMode == g.ModeProgramSearch {
		fyne.Do(func() {
			InputEntry.SetPlaceHolder("Program search...")
		})
		updateContent(nil)
	}
	if newMode == g.ModeDocumentSearch {
		fyne.Do(func() {
			if g.FinishedCachingDocs {
				InputEntry.SetPlaceHolder("Document search...")
			} else {
				InputEntry.SetPlaceHolder("Document search [still caching]...")
			}
		})
		updateContent(nil)
	}
}

func ShowWindow() {
	g.Shown = true
	g.CurrentMode = g.ModeProgramSearch
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
