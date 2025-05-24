/*
	Outstanding issues:

	- Still need an icon and not the example .ico file I found somewhere
	- What is a "gizmo"?
	- How should I handle the hotkeys really?
	- How should I handle hiding a window really?
	-
*/

package main

import (
	_ "embed"
	"github.com/AllenDang/giu"
	"github.com/getlantern/systray"
	"github.com/lxn/win"
	"github.com/robotn/gohook"
	"golang.org/x/sys/windows"
	"log"
	"os"
)

var (
	window   *giu.MasterWindow
	shown    bool          = false
	wasShown bool          = false
	quit     chan struct{} = make(chan struct{})

	//go:embed assets/icon.ico
	iconBytes []byte
)

func main() {
	go systray.Run(onReady, onExit)

	window = giu.NewMasterWindow("winfastnav", 400, 300,
		giu.MasterWindowFlagsFrameless|giu.MasterWindowFlagsNotResizable)

	window.SetPos(-2000, -2000)
	window.SetSize(0, 0)
	window.SetTargetFPS(30) // Need to figure out what I want here. maybe hz / 2?

	// Should never 'close' unless from the systray button
	window.SetCloseCallback(func() bool {
		shown = false
		return false
	})

	go listenHotkeys()
	// This doesn't work.
	// hideFromTaskbar("winnavbar")
	// hideFromTaskbar("gizmo")

	window.Run(loop)
}

func listenHotkeys() {
	// Apparently "cmd" is the Windows key. Huh.
	hook.Register(hook.KeyDown, []string{"alt", "o"}, func(e hook.Event) {
		shown = !shown
		log.Printf("Hotkey pressed, shown: %v", shown)
	})

	s := hook.Start()
	defer hook.End()
	<-hook.Process(s)
}

func focusWindow() {
	titlePtr, err := windows.UTF16PtrFromString("winfastnav")
	if err != nil {
		return
	}
	hwnd := win.FindWindow(nil, titlePtr)
	if hwnd != 0 {
		win.ShowWindow(hwnd, win.SW_SHOWNORMAL)
		win.SetForegroundWindow(hwnd)
	}
}

func hideFromTaskbar(windowName string) {
	titlePtr, err := windows.UTF16PtrFromString(windowName)
	if err != nil {
		return
	}
	hwnd := win.FindWindow(nil, titlePtr)
	if hwnd == 0 {
		return
	}

	style := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
	style &^= win.WS_EX_APPWINDOW // Remove WS_EX_APPWINDOW
	style |= win.WS_EX_TOOLWINDOW // Add WS_EX_TOOLWINDOW
	win.SetWindowLong(hwnd, win.GWL_EXSTYLE, style)

	win.SetWindowPos(
		hwnd,
		win.HWND_TOPMOST,
		0, 0, 0, 0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE|win.SWP_FRAMECHANGED,
	)
}

func onReady() {
	systray.SetIcon(getIcon())
	systray.SetTitle("winfastnav")
	systray.SetTooltip("A fast Windows navigation tool")

	mToggle := systray.AddMenuItem("Show", "Show window")
	mQuit := systray.AddMenuItem("Exit", "Exit program")

	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				shown = !shown
			case <-mQuit.ClickedCh:
				systray.Quit()
				os.Exit(0)
			case <-quit:
				return
			}
		}
	}()
}

func onExit() {
	close(quit)
}

func loop() {
	if !shown {
		window.SetPos(-2000, -2000)
		window.SetSize(0, 0)
		window.SetTargetFPS(1) // Not sure this really helps anything
		wasShown = false
		return
	}

	window.SetPos(100, 100)
	window.SetSize(400, 300)

	if !wasShown {
		focusWindow()
		wasShown = true
	}

	giu.SingleWindow().Layout(
		giu.Label("Hello world"),
		giu.Button("Hide Window").OnClick(func() {
			shown = false
		}),
		giu.Separator(),
		giu.Label("Alt+O to summon, ESC to hide."),
	)

	if giu.IsKeyPressed(giu.KeyEscape) {
		shown = false
	}
}

func getIcon() []byte {
	return iconBytes
}
