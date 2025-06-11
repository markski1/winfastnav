package core

import (
	"fmt"
	"strings"
	"winfastnav/internal/documents"

	"winfastnav/internal/apps"
	"winfastnav/internal/globals"
	"winfastnav/internal/settings"
	"winfastnav/internal/utils"
)

func HandleTextInput(query string) (retItems []globals.Resource, resultStr *string) {
	if len(query) == 0 {
		return nil, nil
	}

	// math evaluation
	if utils.IsMath(query) {
		expr := strings.ReplaceAll(query, " ", "")
		if val, err := utils.EvalMath(expr); err == nil {
			return nil, &val
		}
	}

	switch globals.CurrentMode {
	case globals.ModeSearchInternet:
		s := fmt.Sprintf("Internet search: %s", query)
		s = utils.WrapTextByWords(s, 64)
		return nil, &s

	case globals.ModeSearchProgram:
		findItems := apps.FindAppResults(query)
		return findItems, nil

	case globals.ModeSearchDocument:
		findItems := documents.FilterDocumentsByName(query)
		return findItems, nil

	case globals.ModeAskGPT:
		s := fmt.Sprintf("Quick GPT: %s", query)
		s = utils.WrapTextByWords(s, 64)
		return nil, &s
	}

	return nil, nil
}

// UpdateSearchSetting updates the saved search-string.
func UpdateSearchSetting(s string) {
	globals.SearchString = s
	_ = settings.SetSetting("searchstring", s)
}
