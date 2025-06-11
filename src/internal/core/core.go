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

	// internet search
	if query[0] == '@' {
		s := fmt.Sprintf("Internet search: %s", query[1:])
		s = utils.WrapTextByWords(s, 64)
		return nil, &s
	}

	if utils.StartsWith(query, "!") {
		s := fmt.Sprintf("QuickGPT: %s", query[1:])
		s = utils.WrapTextByWords(s, 64)
		return nil, &s
	}

	// math evaluation
	if utils.IsMath(query) {
		expr := strings.ReplaceAll(query, " ", "")
		if val, err := utils.EvalMath(expr); err == nil {
			return nil, &val
		}
	}

	if globals.CurrentMode == globals.ModeProgramSearch {
		findItems := apps.FindAppResults(query)
		return findItems, nil
	} else if globals.CurrentMode == globals.ModeDocumentSearch {
		findItems := documents.FilterDocumentsByName(query)
		return findItems, nil
	}
	return nil, nil
}

// UpdateSearchSetting updates the saved search-string.
func UpdateSearchSetting(s string) {
	globals.SearchString = s
	_ = settings.SetSetting("searchstring", s)
}
