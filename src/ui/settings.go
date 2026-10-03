package ui

import (
	"path/filepath"
	"slices"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"winfastnav/internal/apps"
	"winfastnav/internal/core"
	"winfastnav/internal/documents"
	g "winfastnav/internal/globals"
	"winfastnav/internal/utils"
)

type settingsPane uint8

const (
	settingsGeneral settingsPane = iota
	settingsAppearance
	settingsIndexing
	settingsHiddenApps
	settingsPaneCount
)

var settingsPaneTitles = [settingsPaneCount]string{"General", "Appearance", "Indexing", "Hidden apps"}

func (l *launcher) settingsPage(gtx layout.Context) layout.Dimensions {
	for {
		e, ok := l.settings.Update(gtx)
		if !ok {
			break
		}
		if _, changed := e.(widget.ChangeEvent); changed && l.settings.Text() != g.SearchString {
			if err := core.UpdateSearchSetting(l.settings.Text()); err != nil {
				l.settingsStatus = "Could not save search URL: " + err.Error()
			} else {
				l.settingsStatus = "Search URL saved."
			}
		}
	}
	for _, editor := range []*widget.Editor{&l.indexRoots, &l.indexExclusions} {
		for {
			_, ok := editor.Update(gtx)
			if !ok {
				break
			}
		}
	}
	for pane := range l.settingsNav {
		for l.settingsNav[pane].Clicked(gtx) {
			if l.settingsPane != settingsPane(pane) {
				l.settingsPane = settingsPane(pane)
				l.settingsList.Position = layout.Position{}
			}
		}
	}
	if l.startupSwitch.Update(gtx) {
		l.setStartup(l.startupSwitch.Value)
	}
	for index := range l.themeChoices {
		for l.themeChoices[index].Clicked(gtx) {
			if l.lightTheme != (index == 1) {
				l.toggleTheme()
			}
		}
		for l.densityChoices[index].Clicked(gtx) {
			if l.compact != (index == 1) {
				l.toggleDensity()
			}
		}
	}
	for index := range l.textSizeChoices {
		for l.textSizeChoices[index].Clicked(gtx) {
			l.setTextSize(index)
		}
	}
	for l.reindex.Clicked(gtx) {
		l.commitIndexSettings(true)
	}
	for _, path := range apps.BlockedApplications() {
		for l.unblockButton(path).Clicked(gtx) {
			if err := apps.UnblockApplication(path); err != nil {
				l.settingsStatus = "Could not restore app: " + err.Error()
			} else {
				l.settingsStatus = "App restored to search."
			}
		}
	}

	return l.panePage(gtx, "Settings", l.settingsSidebar, &l.settingsList, l.settingsSaveStatus(), l.settingsPaneWidgets()...)
}

func (l *launcher) settingsSidebar(gtx layout.Context) layout.Dimensions {
	return l.paneSidebar(gtx, settingsPaneTitles[:], l.settingsNav[:], int(l.settingsPane))
}

func (l *launcher) settingsPaneWidgets() []layout.Widget {
	header := func(title, description string) layout.Widget {
		return inset(layout.Inset{Bottom: unit.Dp(20)}, column(unit.Dp(6), l.title(title, unit.Sp(16)), l.description(description)))
	}
	switch l.settingsPane {
	case settingsAppearance:
		return []layout.Widget{
			header("Appearance", "Make the launcher feel right for you."),
			func(gtx layout.Context) layout.Dimensions {
				return l.settingsChoices(gtx, "Theme", "Choose a light or dark background.", &l.themeChoices, [2]string{"Dark", "Light"}, l.lightTheme)
			},
			func(gtx layout.Context) layout.Dimensions {
				return l.settingsChoices(gtx, "Density", "Adjust spacing in search results and controls.", &l.densityChoices, [2]string{"Comfortable", "Compact"}, l.compact)
			},
			l.textSizeControls,
		}
	case settingsIndexing:
		return []layout.Widget{
			header("Indexing", "Choose where to look for documents."),
			func(gtx layout.Context) layout.Dimensions {
				return l.settingsField(gtx, "Search folders", "Separate folders with semicolons.", `%USERPROFILE%\Documents; D:\Projects`, &l.indexRoots)
			},
			func(gtx layout.Context) layout.Dimensions {
				return l.settingsField(gtx, "Excluded folders", "Use folder names or full paths, separated by semicolons.", `node_modules; venv; C:\Temp\Archive`, &l.indexExclusions)
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = 0
					return l.button(gtx, &l.reindex, "Re-index now")
				})
			},
			l.description("Re-index applies folder changes immediately."),
		}
	case settingsHiddenApps:
		widgets := []layout.Widget{header("Hidden apps", "Restore apps you’ve hidden from search.")}
		paths := apps.BlockedApplications()
		if len(paths) == 0 {
			return append(widgets, column(unit.Dp(8), l.title("No hidden apps", unit.Sp(13)),
				l.description("Press Delete on an app in search results to hide it. You can restore it here later.")))
		}
		for _, path := range paths {
			widgets = append(widgets, func(gtx layout.Context) layout.Dimensions { return l.blockedAppRow(gtx, path) })
		}
		return widgets
	default:
		return []layout.Widget{
			header("General", "Web search and Windows sign-in."),
			func(gtx layout.Context) layout.Dimensions {
				return l.settingsField(gtx, "Web search URL", "Use %s where the search query should go.", "https://duckduckgo.com/?q=%s", &l.settings)
			},
			column(unit.Dp(6), l.title("Launch at sign-in", unit.Sp(12)),
				l.description("Start winfastnav automatically when you sign in."),
				inset(layout.Inset{Top: unit.Dp(4)}, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Switch(l.theme, &l.startupSwitch, "Launch at sign-in").Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := "Off"
							if l.startupEnabled {
								label = "On"
							}
							return l.description(label)(gtx)
						}),
					)
				})),
		}
	}
}

func (l *launcher) settingsField(gtx layout.Context, title, description, hint string, state *widget.Editor) layout.Dimensions {
	editor := material.Editor(l.theme, state, hint)
	editor.TextSize = unit.Sp(13)
	editor.Color = l.palette.text
	editor.HintColor = l.palette.muted
	return inset(layout.Inset{Bottom: unit.Dp(20)}, column(0,
		l.title(title, unit.Sp(12)),
		inset(layout.Inset{Top: unit.Dp(8)}, func(gtx layout.Context) layout.Dimensions { return l.input(gtx, editor.Layout) }),
		inset(layout.Inset{Top: unit.Dp(6)}, l.description(description)),
	))(gtx)
}

func (l *launcher) settingsChoices(gtx layout.Context, title, description string, choices *[2]widget.Clickable, labels [2]string, secondSelected bool) layout.Dimensions {
	selected := 0
	if secondSelected {
		selected = 1
	}
	return inset(layout.Inset{Bottom: unit.Dp(16)}, column(unit.Dp(6),
		l.title(title, unit.Sp(12)), l.description(description),
		inset(layout.Inset{Top: unit.Dp(4)}, l.choiceGroup(choices[:], labels[:], selected)),
	))(gtx)
}

func (l *launcher) setStartup(enabled bool) {
	set := l.startupSet
	if set == nil {
		set = utils.SetStartupEnabled
	}
	if err := set(enabled); err != nil {
		l.startupSwitch.Value = l.startupEnabled
		l.settingsStatus = "Could not change launch at sign-in: " + err.Error()
		return
	}
	l.startupEnabled, l.startupSwitch.Value = enabled, enabled
	l.settingsStatus = "Launch at sign-in saved."
}

func (l *launcher) indexSettingsPending() bool {
	return !slices.EqualFunc(documents.ParseIndexList(l.indexRoots.Text()), l.savedIndexConfig.Roots, strings.EqualFold) ||
		!slices.EqualFunc(documents.ParseIndexList(l.indexExclusions.Text()), l.savedIndexConfig.Exclusions, strings.EqualFold)
}

func (l *launcher) settingsSaveStatus() string {
	status := l.settingsStatus
	if l.indexSettingsPending() {
		if status != "" {
			status += "\n"
		}
		return status + "Folder changes pending. Apply on exit, or re-index now."
	}
	if status == "" {
		return "All changes saved. Esc returns to Menu."
	}
	return status
}

func (l *launcher) unblockButton(path string) *widget.Clickable {
	if l.unblockButtons == nil {
		l.unblockButtons = make(map[string]*widget.Clickable)
	}
	button := l.unblockButtons[path]
	if button == nil {
		button = new(widget.Clickable)
		l.unblockButtons[path] = button
	}
	return button
}

func (l *launcher) blockedAppRow(gtx layout.Context, path string) layout.Dimensions {
	return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return l.surface(gtx, l.palette.row, func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, column(unit.Dp(4),
						func(gtx layout.Context) layout.Dimensions {
							return l.resultText(gtx, filepath.Base(path), unit.Sp(12), l.palette.text)
						},
						func(gtx layout.Context) layout.Dimensions {
							style := material.Label(l.theme, unit.Sp(10.5), path)
							style.Color = l.palette.secondary
							style.MaxLines = 2
							return style.Layout(gtx)
						},
					)),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return l.button(gtx, l.unblockButton(path), "Restore") }),
				)
			})
		})
	})
}
