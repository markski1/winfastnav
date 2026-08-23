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
		switch globals.CurrentMode {
		case globals.ModeSearchProgram:
			return apps.RecentApplications(), nil
		case globals.ModeSearchDocument:
			return documents.RecentDocuments(), nil
		}
		return nil, nil
	}

	calculation, calculated := calculate(query)
	if strings.HasPrefix(strings.TrimSpace(query), "=") && !calculated {
		help := "Enter a mathematical expression (2+2) or unit to convert (20in).\n\n" +
			"Supported units: Weight, length, speed and temperature.\n" +
			"Supported operators: +, -, *, /"
		return nil, &help
	}

	switch globals.CurrentMode {
	case globals.ModeSearchInternet:
		s := fmt.Sprintf("Internet search: %s", query)
		s = utils.WrapTextByWords(s, 64)
		return nil, &s

	case globals.ModeSearchProgram:
		findItems := apps.FindAppResults(query)
		return withCalculation(findItems, calculation, calculated), nil

	case globals.ModeSearchDocument:
		findItems := documents.FilterDocumentsByName(query)
		return withCalculation(findItems, calculation, calculated), nil

	case globals.ModeAskGPT:
		s := fmt.Sprintf("Quick GPT: %s", query)
		s = utils.WrapTextByWords(s, 64)
		return nil, &s
	}

	return nil, nil
}

func calculate(query string) (string, bool) {
	expression := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(query), "="))
	if expression == "" {
		return "", false
	}
	if utils.IsMath(expression) {
		result, err := utils.EvalMath(strings.ReplaceAll(expression, " ", ""))
		return result, err == nil
	}
	if result := utils.ConvertUnit(expression); result != "" {
		return result, true
	}
	return "", false
}

func withCalculation(resources []globals.Resource, calculation string, calculated bool) []globals.Resource {
	if !calculated {
		return resources
	}
	display := strings.ReplaceAll(calculation, "\n", "  /  ")
	result := make([]globals.Resource, 0, len(resources)+1)
	result = append(result, globals.Resource{Name: display, Computed: true})
	return append(result, resources...)
}

// UpdateSearchSetting updates the saved search-string.
func UpdateSearchSetting(s string) {
	globals.SearchString = s
	_ = settings.SetSetting("searchstring", s)
}
