/*
	CustomEntry

	Entry widget, but calls the given function when the down arrow is pressed.
	Used to focus the result list when arrow down is pushed.
*/

package widgets

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"log"
	"net/url"
	"os/exec"
	g "winfastnav/internal/globals"
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
				if len(e.Text) > 0 && e.Text[0] == '@' {
					openURI(fmt.Sprintf(g.SearchString, url.QueryEscape(e.Text[1:])))
					g.NavWindow.Hide()
					return
				}
			})
		}
	}

	// Call the parent's TypedKey for normal behavior
	e.Entry.TypedKey(key)
}

func openURI(uri string) {
	log.Printf(uri)
	cmd := exec.Command("cmd", "/c", "start", uri)
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to open URI: %v", err)
	}
}
