package globals

import (
	_ "embed"
)

type Resource struct {
	Name     string
	Filepath string
	Computed bool
	Document bool
}

const (
	ModeSearchProgram  = 10
	ModeSearchInternet = 12

	ModeAskGPT = 31
)

var (
	AppName       = "winfastnav v0.6"
	AppList       []Resource
	ExecBlocklist []string
	SearchString  string

	FinishedCachingDocs = false

	CurrentMode int = ModeSearchProgram

	//go:embed assets/icon.ico
	IconBytes []byte
)
