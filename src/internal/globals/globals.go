package globals

import (
	_ "embed"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

type App struct {
	Name     string
	ExecPath string
}

var (
	AppList       []App
	ExecBlocklist []string
	SearchString  string

	NavApp    = app.New()
	NavWindow fyne.Window
	Shown     bool = false

	//go:embed assets/icon.ico
	IconBytes []byte
)
