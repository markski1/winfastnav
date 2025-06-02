/*
	CustomList

	Because the List widget from Fyne doesn't support keyboard selection,
	nor activating by pushing enter, we have to roll our own.

	Rather than allow selections like the original List widget, this is simply
	a list of buttons that can be activated by pressing enter or being clicked.

	We do a few things to avoid too many allocations, namely having a fixed count
	of reusable buttons and the component being updatable instead of spawning a
	new list each time as with the default List widget.
*/

package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	d "winfastnav/assets"
)

const maxButtons = 20

type CustomList struct {
	widget.BaseWidget

	Items      []d.App
	OnSelected func(index int, item d.App)

	selectedIndex int
	renderer      *customListRenderer
}

func NewCustomList(items []d.App, onSelected func(index int, item d.App)) *CustomList {
	sl := &CustomList{Items: items, OnSelected: onSelected}
	sl.selectedIndex = -1
	sl.ExtendBaseWidget(sl)
	return sl
}

func (sl *CustomList) CreateRenderer() fyne.WidgetRenderer {
	buttons := make([]*widget.Button, maxButtons)

	for i := 0; i < maxButtons; i++ {
		idx := i
		var btn *widget.Button
		if i < len(sl.Items) {
			app := sl.Items[i]
			btn = widget.NewButton(app.Name, func() {
				sl.selectedIndex = idx
				sl.activateSelection()
			})
		} else {
			btn = widget.NewButton("", nil)
			btn.Hide()
		}
		buttons[i] = btn
	}

	objects := make([]fyne.CanvasObject, maxButtons)
	for i, btn := range buttons {
		objects[i] = btn
	}

	content := container.NewVBox(objects...)
	scroll := container.NewVScroll(content)

	r := &customListRenderer{
		list:    sl,
		buttons: buttons,
		content: content,
		scroll:  scroll,
	}
	sl.renderer = r
	return r
}

func (sl *CustomList) UpdateItems(newItems []d.App) {
	sl.Items = newItems
	sl.selectedIndex = -1
	if sl.renderer == nil {
		sl.Refresh()
		return
	}
	r := sl.renderer

	for i, btn := range r.buttons {
		if i < len(newItems) {
			btn.SetText(newItems[i].Name)
			btn.Importance = widget.MediumImportance
			btn.Show()
		} else {
			btn.Hide()
		}
	}

	r.Refresh()
}

func (sl *CustomList) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyUp:
		sl.moveSelection(-1)
	case fyne.KeyDown:
		sl.moveSelection(1)
	case fyne.KeyReturn, fyne.KeyEnter:
		sl.activateSelection()
	case fyne.KeyDelete:
		itemName := sl.Items[sl.selectedIndex].Name
		dlg := dialog.NewConfirm("Hide app",
			"Are you sure you want to hide \""+itemName+"\"?",
			func(confirmed bool) {
				if confirmed {
					d.BlockApplication(sl.Items[sl.selectedIndex])
					NavWindow.Canvas().Focus(InputEntry)
					InputEntry.SetText("")
				}
			}, fyne.CurrentApp().Driver().AllWindows()[0])
		dlg.Show()
	}
}

// TypedRune no-op: Even if we don't really care abt input, Fyne requires
// this for something to be focusable.
func (sl *CustomList) TypedRune(r rune) {
	_ = r
}

func (sl *CustomList) FocusGained() {
	if sl.selectedIndex < 0 && len(sl.Items) > 0 {
		// select first item as soon as focus is gained
		sl.selectedIndex = 0
	}
	sl.Refresh()
}

func (sl *CustomList) FocusLost() {
	sl.Refresh()
}

func (sl *CustomList) moveSelection(delta int) {
	n := len(sl.Items)
	if n == 0 {
		return
	}
	// return focus to the text input if we go up from the top.
	if delta < 0 && sl.selectedIndex == 0 {
		sl.selectedIndex = -1
		fyne.Do(func() {
			NavWindow.Canvas().Focus(InputEntry)
		})
		return
	}
	// wrap around
	sl.selectedIndex = (sl.selectedIndex + delta + n) % n
	sl.scrollToSelection()
	sl.Refresh()
}

func (sl *CustomList) scrollToSelection() {
	if sl.renderer == nil || sl.selectedIndex < 0 || sl.selectedIndex >= len(sl.Items) {
		return
	}

	// Get button height
	if len(sl.renderer.buttons) == 0 {
		return
	}

	// We need to account for padding. Tried a few numbers, 1.12 does well.
	buttonHeight := sl.renderer.buttons[0].MinSize().Height * 1.12

	// Calculate the position of the selected button
	selectedPosition := float32(sl.selectedIndex) * buttonHeight

	scrollHeight := sl.renderer.scroll.Size().Height
	currentOffset := sl.renderer.scroll.Offset.Y

	if selectedPosition < currentOffset {
		sl.renderer.scroll.Offset.Y = selectedPosition
	} else if selectedPosition+buttonHeight > currentOffset+scrollHeight {
		sl.renderer.scroll.Offset.Y = selectedPosition + buttonHeight - scrollHeight
	}

	sl.renderer.scroll.Refresh()
}

func (sl *CustomList) activateSelection() {
	if sl.selectedIndex >= 0 && sl.selectedIndex < len(sl.Items) && sl.OnSelected != nil {
		sl.OnSelected(sl.selectedIndex, sl.Items[sl.selectedIndex])
	}
}

type customListRenderer struct {
	list    *CustomList
	buttons []*widget.Button
	content *fyne.Container
	scroll  *container.Scroll
}

func (r *customListRenderer) Layout(size fyne.Size) {
	r.scroll.Resize(size)
}

func (r *customListRenderer) MinSize() fyne.Size {
	return r.scroll.MinSize()
}

func (r *customListRenderer) Refresh() {
	for i, btn := range r.buttons {
		if i == r.list.selectedIndex {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.MediumImportance
		}
		btn.Refresh()
	}
	r.content.Refresh()
	r.scroll.Refresh()
}

func (r *customListRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.scroll}
}

func (r *customListRenderer) Destroy() {}
