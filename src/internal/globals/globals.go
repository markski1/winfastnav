package globals

import (
	_ "embed"
)

type Resource struct {
	Name       string
	Filepath   string
	Computed   bool
	Document   bool
	WebSearch  string         `json:"-"`
	Assistant  string         `json:"-"`
	Command    *SystemCommand `json:"-"`
	SearchName string         `json:"-"`
	SearchPath string         `json:"-"`
}

type SystemCommand struct {
	Action string
	Detail string
}

var (
	AppName       = "winfastnav v0.7"
	AppList       []Resource
	ExecBlocklist []string
	SearchString  string

	//go:embed assets/icon.ico
	IconBytes []byte
)
