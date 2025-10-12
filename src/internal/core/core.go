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

	// evaluations
	if query[0] == '=' {
		result := ""
		expr := strings.ReplaceAll(query, "=", "")
		// Try to do math eval
		if utils.IsMath(expr) {
			expr := strings.ReplaceAll(expr, " ", "")
			if result, err := utils.EvalMath(expr); err == nil {
				return nil, &result
			}
		}
		// Otherwise try a conversion
		if utils.HasUnit(expr) {
			result = utils.ConvertUnit(expr)
			return nil, &result
		}
		// Otherwise show an explanation
		result = "Enter a mathematical expression (2+2) or unit to convert (20in).\n" +
			"\n" +
			"Supported units: Weight, length, speed and temperature.\n" +
			"Supported operators: +, -, *, /"
		return nil, &result
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
