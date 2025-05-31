/*
	CustomList

	Because the List widget from Fyne doesn't support keyboard selection,
	nor activating by pushing enter, we have to roll our own.

	Rather than allow selections like the original List widget, this is simply
	a list of buttons that can be activated by pressing enter or being clicked.
*/

package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	d "winfastnav/assets"
)

type CustomList struct {
	widget.BaseWidget

	Items      []d.App
	OnSelected func(index int, item d.App)

	selectedIndex int
}

func NewCustomList(items []d.App, onSelected func(index int, item d.App)) *CustomList {
	sl := &CustomList{Items: items, OnSelected: onSelected}
	sl.selectedIndex = -1
	sl.ExtendBaseWidget(sl)
	return sl
}

func (sl *CustomList) CreateRenderer() fyne.WidgetRenderer {
	buttons := make([]*widget.Button, len(sl.Items))
	objs := make([]fyne.CanvasObject, len(sl.Items))
	for i, app := range sl.Items {
		idx := i
		btn := widget.NewButton(app.Name, func() {
			sl.selectedIndex = idx
			sl.activateSelection()
		})
		buttons[i] = btn
		objs[i] = btn
	}
	content := container.NewVBox(objs...)
	scroll := container.NewVScroll(content)

	return &customListRenderer{
		list:    sl,
		buttons: buttons,
		content: content,
		scroll:  scroll,
	}
}

func (sl *CustomList) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyUp:
		sl.moveSelection(-1)
	case fyne.KeyDown:
		sl.moveSelection(1)
	case fyne.KeyReturn, fyne.KeyEnter:
		sl.activateSelection()
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
	// wrap around
	sl.selectedIndex = (sl.selectedIndex + delta + n) % n
	sl.Refresh()
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
