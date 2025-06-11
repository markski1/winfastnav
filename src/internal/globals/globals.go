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
	ModeProgramSearch   = 0
	ModeChoosingProgram = 1
	ModeDocumentSearch  = 2
)

var (
	AppName       = "winfastnav v0.2"
	AppList       []Resource
	ExecBlocklist []string
	SearchString  string

	FinishedCachingDocs = false

	NavApp      = app.New()
	NavWindow   fyne.Window
	Shown       bool = false
	CurrentMode int  = ModeProgramSearch

	//go:embed assets/icon.ico
	IconBytes []byte
)
