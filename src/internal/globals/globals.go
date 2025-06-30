package globals

import (
	_ "embed"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

type Resource struct {
	Name     string
	Filepath string
}

const (
	ModeSearchProgram  = 10
	ModeSearchDocument = 11
	ModeSearchInternet = 12

	ModeChooseProgram = 21

	ModeAskGPT = 31
)

var (
	AppName       = "winfastnav v0.3"
	AppList       []Resource
	ExecBlocklist []string
	SearchString  string

	FinishedCachingDocs = false

	ShowingMain = true

	NavApp      = app.New()
	NavWindow   fyne.Window
	Shown       bool = false
	CurrentMode int  = ModeSearchProgram

	//go:embed assets/icon.ico
	IconBytes []byte
)
