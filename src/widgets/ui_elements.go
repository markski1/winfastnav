package widgets

import (
	"fyne.io/fyne/v2/app"
)

var (
	NavApp     = app.New()
	NavWindow  = NavApp.NewWindow("winfastnav")
	InputEntry *CustomEntry
	ResultList *CustomList
)
