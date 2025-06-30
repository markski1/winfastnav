package ui

/*
	While not a major issue, there IS a lot of business code in here.
	Eventually this should be moved to `core` and elsewhere.
*/

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
	"net/url"
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
	ProgramResultList  *w.CustomList[g.Resource]
	DocumentResultList *w.CustomList[g.Resource]
	openProgramList    *w.CustomList[string]
	inputContainer     *fyne.Container

	okButton = widget.NewButton("OK", func() {
		updateContent(nil)
	})
)

func SetupUI() {
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

	g.NavWindow.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		if key.Name == fyne.KeyReturn || key.Name == fyne.KeyEnter {
			if !g.ShowingMain {
				okButton.OnTapped()
			}
		}
	})

	InputEntry = w.NewCustomEntry(func() {
		fyne.Do(func() {
			if len(InputEntry.Text) > 0 {
				if g.CurrentMode == g.ModeSearchProgram {
					g.NavWindow.Canvas().Focus(ProgramResultList)
				}
				if g.CurrentMode == g.ModeSearchDocument {
					g.NavWindow.Canvas().Focus(DocumentResultList)
				}
			}
			if g.CurrentMode == g.ModeChooseProgram {
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

	ProgramResultList = w.NewCustomList([]g.Resource{}, InputEntry, func(app g.Resource) string { return app.Name }, func(idx int, app g.Resource) {
		err := apps.OpenProgram(app.Filepath)
		if err != nil {
			log.Printf("Error opening program: %v", err)
			MainShowText("Sorry, there was an error opening the program.")
			return
		}
		HideWindow()
	})

	DocumentResultList = w.NewCustomList([]g.Resource{}, InputEntry, func(doc g.Resource) string { return doc.Name }, func(idx int, doc g.Resource) {
		err := documents.OpenFile(doc.Filepath)
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
}

func updateContent(aContent fyne.CanvasObject) {
	g.ShowingMain = true
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
	g.ShowingMain = false

	bottomVBox := container.NewVBox(
		okButton,
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
			ShowHelp()
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
			dlg := dialog.NewConfirm("Clear Blocklist",
				"Are you sure you want to unblock all blocked apps?",
				func(confirmed bool) {
					if confirmed {
						apps.UnblockAllApplications()
						dialog.NewInformation("Blocklist cleared", "All apps have been unblocked", g.NavWindow).Show()
					}
				}, g.NavWindow)
			dlg.Show()
		}),
	)

	startupBox := container.NewVBox(
		widget.NewLabel("Start with Windows"),
		widget.NewLabel("Note: Put the .exe file in it's desired location."),
		widget.NewButton("Add to Startup", func() {
			err := utils.AddToStartup()
			if err != nil {
				MainShowText("Error adding to startup: " + err.Error())
			}
			MainShowText("winfastnav added to startup!")
		}),
	)

	content := container.NewVBox(
		searchStringBox,
		widget.NewSeparator(),
		blocklistBox,
		widget.NewSeparator(),
		startupBox,
	)

	showContent(content)
}

func ShowHelp() {
	first := container.NewVBox(
		widget.NewLabel(
			"Keys:\n" +
				"ALT + O: Summon\n" +
				"ESC: Hide\n" +
				"Delete: Hide app",
		),
	)

	second := container.NewVBox(
		widget.NewRichText(
			&widget.TextSegment{
				Text: "Command modes:\n" +
					":p | Program search (default)\n" +
					":d | Document search\n" +
					":w | Internet search\n" +
					":s | Switch to Window\n" +
					":g | Quick GPT\n" +
					":c | Run command\n" +
					":r | Re-index all resources\n" +
					":x | Quit",
				Style: widget.RichTextStyle{
					TextStyle: fyne.TextStyle{Monospace: true},
				},
			},
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
	if g.CurrentMode == g.ModeChooseProgram {
		return
	}

	listGet, mathResult := core.HandleTextInput(input)
	if mathResult != nil {
		MainShowText(*mathResult)
		return
	}
	if g.CurrentMode == g.ModeSearchProgram {
		ProgramResultList.UpdateItems(listGet)
		updateContent(ProgramResultList)
		return
	} else if g.CurrentMode == g.ModeSearchDocument {
		DocumentResultList.UpdateItems(listGet)
		updateContent(DocumentResultList)
		return
	}
	updateContent(nil)
}

func updateSubmitContent(inputText string) {
	if len(inputText) > 0 {
		if inputText[0] == ':' {
			InputEntry.SetText("")

			switch inputText[1] {
			case 'w':
				SetMode(g.ModeSearchInternet)
			case 'g':
				SetMode(g.ModeAskGPT)
			case 'd':
				SetMode(g.ModeSearchDocument)
			case 'p':
				SetMode(g.ModeSearchProgram)
			case 's':
				SetMode(g.ModeChooseProgram)
			case 'h':
				ShowHelp()
				return
			case 'x':
				g.NavApp.Quit()
			case 'r':
				MainShowText("Re-indexing programs and documents.")
				go documents.SetupDocs()
				go apps.SetupApps()
			case 'q':
				HideWindow()
			}
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
		if g.CurrentMode == g.ModeAskGPT {
			MainShowText("Please wait...")
			go func(p string) {
				result := utils.MakeGPTReq(p)
				fyne.Do(func() {
					MainShowText(result)
				})
			}(inputText)
			return
		}
		if g.CurrentMode == g.ModeChooseProgram {
			num, err := strconv.Atoi(inputText)
			if err == nil {
				HideWindow()
				apps.FocusWindow(num)
				return
			}
		}
		if g.CurrentMode == g.ModeSearchInternet {
			err := utils.OpenURI(fmt.Sprintf(g.SearchString, url.QueryEscape(InputEntry.Text)))
			if err == nil {
				HideWindow()
			} else {
				MainShowText("Sorry, there was an error opening your web browser.")
			}
			return
		}
		g.NavWindow.Canvas().Focus(ProgramResultList)
	}
	// Otherwise attempt to focus the list.
	if g.CurrentMode == g.ModeChooseProgram {
		g.NavWindow.Canvas().Focus(openProgramList)
		return
	}
}

func SetMode(newMode int) {
	g.CurrentMode = newMode

	switch newMode {
	case g.ModeChooseProgram:
		fyne.Do(func() {
			InputEntry.SetPlaceHolder("Choose window...")
		})
		openProgramList.UpdateItems(apps.GetOpenWindows())
		updateContent(openProgramList)

	case g.ModeSearchProgram:
		fyne.Do(func() {
			InputEntry.SetPlaceHolder("Program search...")
		})
		updateContent(nil)

	case g.ModeSearchDocument:
		placeholder := "Document search..."
		if !g.FinishedCachingDocs {
			placeholder = "Document search [still caching]..."
		}
		fyne.Do(func() {
			InputEntry.SetPlaceHolder(placeholder)
		})
		updateContent(nil)

	case g.ModeSearchInternet:
		fyne.Do(func() {
			InputEntry.SetPlaceHolder("Internet search...")
		})
		updateContent(nil)

	case g.ModeAskGPT:
		fyne.Do(func() {
			InputEntry.SetPlaceHolder("Quick GPT...")
		})
		updateContent(nil)
	}
}

func ShowWindow() {
	g.Shown = true
	g.CurrentMode = g.ModeSearchProgram
	fyne.Do(func() {
		g.NavWindow.Show()
		InputEntry.SetPlaceHolder("Program search...")
		MainShowText(g.AppName + "\nEnter :h for help.")
		for i := 0; i < 3; i++ {
			time.Sleep(33 * time.Millisecond)
			g.NavWindow.RequestFocus()
			g.NavWindow.Canvas().Focus(InputEntry)
		}
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
