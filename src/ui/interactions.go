package ui

import (
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	appsettings "winfastnav/internal/settings"
)

var textSizeSettings = [3]string{"default", "large", "larger"}

func textSizeIndexForSetting(value string) int {
	for index, setting := range textSizeSettings {
		if strings.EqualFold(value, setting) {
			return index
		}
	}
	return 0
}

func textSizeScale(index int) float32 {
	if index < 0 || index >= len(textSizeSettings) {
		return 1
	}
	return [3]float32{1, 1.15, 1.3}[index]
}

func effectiveSpScale(scale float32) float32 {
	if scale == 0 {
		return 1
	}
	return scale
}

func (l *launcher) setTextSize(index int) {
	if index < 0 || index >= len(textSizeSettings) || l.textSizeIndex == index {
		return
	}
	if err := appsettings.SetSetting("textsize", textSizeSettings[index]); err != nil {
		l.settingsStatus = "Could not save text size: " + err.Error()
		return
	}
	l.textSizeIndex = index
	l.settingsStatus = "Text size saved."
}

func (l *launcher) prepareShow() {
	l.clearItems()
	l.updateState(func(state *uiState) {
		state.visible = true
		state.page = pageLauncher
		state.actionMenu = false
		state.focusSearch = true
		if !state.answer {
			state.query, state.message = "", ""
			state.loading = false
			state.resultCount, state.selected = 0, -1
		}
	})
}

func (l *launcher) prepareHide() {
	l.clearItems()
	l.updateState(func(state *uiState) {
		state.visible = false
		state.focusSearch = false
		state.actionMenu = false
		if !state.answer {
			l.answerGeneration.Add(1)
			state.query, state.message = "", ""
			state.loading = false
			state.resultCount, state.selected = 0, -1
		}
	})
}

func (l *launcher) backPage() {
	state := l.snapshot()
	if state.page == pageSettings && !l.commitIndexSettings(false) {
		return
	}
	switch state.page {
	case pageSettings, pageHelp, pageAbout:
		l.updateState(func(state *uiState) {
			state.page = pageMenu
			state.focusSearch = false
		})
	default:
		l.launcher()
	}
}

func (l *launcher) textSizeControls(gtx layout.Context) layout.Dimensions {
	return column(unit.Dp(8), l.title("Text size", unit.Sp(12)), l.choiceGroup(l.textSizeChoices[:], []string{"Default", "Large", "Larger"}, l.textSizeIndex))(gtx)
}
