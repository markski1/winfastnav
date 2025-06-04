/*
	CustomList

	Because the List widget from Fyne doesn't support keyboard selection,
	nor activating by pushing enter, we have to roll our own.

	Rather than allow selections like the original List widget, this is simply
	a list of labels that can be activated by pressing enter.

	We do a few things to avoid too many allocations, namely having a fixed count
	of reusable labels and the component being updatable instead of spawning a
	new list each time as with the default List widget.
*/

package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"winfastnav/internal/apps"
	g "winfastnav/internal/globals"
)

const maxItems = 30

type listItem struct {
	label      *widget.Label
	background *canvas.Rectangle
	container  *fyne.Container
}

type CustomList[T any] struct {
	widget.BaseWidget

	Items       []T
	OnSelected  func(index int, item T)
	DisplayFunc func(T) string // Function to extract display text from item

	selectedIndex int
	renderer      *customListRenderer[T]
	input         *CustomEntry
}

func NewCustomList[T any](items []T, inputRef *CustomEntry, displayFunc func(T) string, onSelected func(index int, item T)) *CustomList[T] {
	sl := &CustomList[T]{
		Items:       items,
		OnSelected:  onSelected,
		DisplayFunc: displayFunc,
	}
	sl.selectedIndex = -1
	sl.ExtendBaseWidget(sl)
	sl.input = inputRef
	return sl
}

func (sl *CustomList[T]) CreateRenderer() fyne.WidgetRenderer {
	items := make([]*listItem, maxItems)

	for i := 0; i < maxItems; i++ {
		label := widget.NewLabel("")
		label.Alignment = fyne.TextAlignLeading

		background := canvas.NewRectangle(color.Transparent)

		itemContainer := container.NewBorder(nil, nil, nil, nil, background, label)

		items[i] = &listItem{
			label:      label,
			background: background,
			container:  itemContainer,
		}

		if i < len(sl.Items) {
			label.SetText(sl.DisplayFunc(sl.Items[i]))
			itemContainer.Show()
		} else {
			itemContainer.Hide()
		}
	}

	objects := make([]fyne.CanvasObject, maxItems)
	for i, item := range items {
		objects[i] = item.container
	}

	content := container.NewVBox(objects...)
	scroll := container.NewVScroll(content)

	r := &customListRenderer[T]{
		list:    sl,
		items:   items,
		content: content,
		scroll:  scroll,
	}
	sl.renderer = r
	return r
}

func (sl *CustomList[T]) UpdateItems(newItems []T) {
	sl.Items = newItems
	sl.selectedIndex = -1
	if sl.renderer == nil {
		sl.Refresh()
		return
	}
	r := sl.renderer

	for i, item := range r.items {
		if i < len(newItems) {
			item.label.SetText(sl.DisplayFunc(newItems[i]))
			item.container.Show()
		} else {
			item.container.Hide()
		}
	}

	r.Refresh()
}

func (sl *CustomList[T]) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyUp:
		sl.moveSelection(-1)
	case fyne.KeyDown:
		sl.moveSelection(1)
	case fyne.KeyReturn, fyne.KeyEnter:
		sl.activateSelection()
	case fyne.KeyDelete:
		// Only for apps!
		if any(sl.Items[sl.selectedIndex]).(g.App) != (g.App{}) {
			app := any(sl.Items[sl.selectedIndex]).(g.App)
			itemName := app.Name
			dlg := dialog.NewConfirm("Hide app",
				"Are you sure you want to hide \""+itemName+"\"?",
				func(confirmed bool) {
					if confirmed {
						apps.BlockApplication(app)
						g.NavWindow.Canvas().Focus(sl.input)
						sl.input.SetText("")
					}
				}, fyne.CurrentApp().Driver().AllWindows()[0])
			dlg.Show()
		}
	}
}

// TypedRune no-op: Even if we don't really care abt input, Fyne requires
// this for something to be focusable.
func (sl *CustomList[T]) TypedRune(r rune) {
	_ = r
}

func (sl *CustomList[T]) FocusGained() {
	if sl.selectedIndex < 0 && len(sl.Items) > 0 {
		// select first item as soon as focus is gained
		sl.selectedIndex = 0
	}
	sl.Refresh()
}

func (sl *CustomList[T]) FocusLost() {
	sl.Refresh()
}

func (sl *CustomList[T]) moveSelection(delta int) {
	n := len(sl.Items)
	if n == 0 {
		return
	}
	// return focus to the text input if we go up from the top.
	if delta < 0 && sl.selectedIndex == 0 {
		sl.selectedIndex = -1
		fyne.Do(func() {
			g.NavWindow.Canvas().Focus(sl.input)
		})
		return
	}
	// wrap around
	sl.selectedIndex = (sl.selectedIndex + delta + n) % n
	sl.scrollToSelection()
	sl.Refresh()
}

func (sl *CustomList[T]) scrollToSelection() {
	if sl.renderer == nil || sl.selectedIndex < 0 || sl.selectedIndex >= len(sl.Items) {
		return
	}

	if len(sl.renderer.items) == 0 {
		return
	}

	// Get item height, accountingfor padding. Tried a few numbers, 1.065 does well.
	itemHeight := sl.renderer.items[0].container.MinSize().Height * 1.065

	selectedPosition := float32(sl.selectedIndex) * itemHeight
	scrollHeight := sl.renderer.scroll.Size().Height
	currentOffset := sl.renderer.scroll.Offset.Y

	if selectedPosition < currentOffset {
		sl.renderer.scroll.Offset.Y = selectedPosition
	} else if selectedPosition+itemHeight > currentOffset+scrollHeight {
		sl.renderer.scroll.Offset.Y = selectedPosition + itemHeight - scrollHeight
	}

	sl.renderer.scroll.Refresh()
}

func (sl *CustomList[T]) activateSelection() {
	if sl.selectedIndex >= 0 && sl.selectedIndex < len(sl.Items) && sl.OnSelected != nil {
		sl.OnSelected(sl.selectedIndex, sl.Items[sl.selectedIndex])
	}
}

type customListRenderer[T any] struct {
	list    *CustomList[T]
	items   []*listItem
	content *fyne.Container
	scroll  *container.Scroll
}

func (r *customListRenderer[T]) Layout(size fyne.Size) {
	r.scroll.Resize(size)
}

func (r *customListRenderer[T]) MinSize() fyne.Size {
	return r.scroll.MinSize()
}

func (r *customListRenderer[T]) Refresh() {
	for i, item := range r.items {
		if i == r.list.selectedIndex {
			// Selected item gets highlighted background
			item.background.FillColor = theme.Color("primary")
		} else {
			// Non-selected items have transparent background
			item.background.FillColor = color.Transparent
		}
		item.background.Refresh()
	}
	r.content.Refresh()
	r.scroll.Refresh()
}

func (r *customListRenderer[T]) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.scroll}
}

func (r *customListRenderer[T]) Destroy() {}
