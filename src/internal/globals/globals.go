package globals

import (
	_ "embed"
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
	AppName       = "winfastnav v0.5"
	AppList       []Resource
	ExecBlocklist []string
	SearchString  string

	FinishedCachingDocs = false

	CurrentMode int = ModeSearchProgram

	//go:embed assets/icon.ico
	IconBytes []byte
)
