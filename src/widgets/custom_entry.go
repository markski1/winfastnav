/*
	CustomEntry

	Entry widget, but calls the given function when the down arrow is pressed.
	Used to focus the result list when arrow down is pushed.
*/

package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type CustomEntry struct {
	widget.Entry
	onDownArrow func()
}

func NewCustomEntry(onDownArrow func()) *CustomEntry {
	entry := &CustomEntry{onDownArrow: onDownArrow}
	entry.ExtendBaseWidget(entry)
	return entry
}

func (e *CustomEntry) TypedKey(key *fyne.KeyEvent) {
	if e.onDownArrow != nil {
		if key.Name == fyne.KeyDown {
			e.onDownArrow()
			return
		}

		if key.Name == fyne.KeyReturn || key.Name == fyne.KeyEnter {
			fyne.Do(func() {
				NavWindow.Canvas().Focus(ResultList)
			})
		}
	}

	// Call the parent's TypedKey for normal behavior
	e.Entry.TypedKey(key)
}
