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

type Document struct {
	Filename string
	Path     string
}

const (
	ModeProgramSearch   = 0
	ModeChoosingProgram = 1
	ModeDocumentSearch  = 2
)

var (
	AppName       = "winfastnav v0.1"
	AppList       []App
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
