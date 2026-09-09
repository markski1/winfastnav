package core

import (
	"fmt"
	"strings"
	"winfastnav/internal/documents"

	"winfastnav/internal/apps"
	"winfastnav/internal/globals"
	"winfastnav/internal/settings"
	"winfastnav/internal/systemactions"
	"winfastnav/internal/utils"
)

const (
	maxCombinedResults = 30
	maxAppResults      = 20
	maxCommandResults  = 3
)

func HandleTextInput(query string) (retItems []globals.Resource, resultStr *string) {
	return HandleTextInputMode(query, globals.CurrentMode)
}

func HandleTextInputMode(query string, mode int) (retItems []globals.Resource, resultStr *string) {
	if len(query) == 0 {
		switch mode {
		case globals.ModeSearchProgram:
			return combineResults(apps.RecentApplications(), nil, documents.RecentDocuments()), nil
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

	switch mode {
	case globals.ModeSearchInternet:
		s := fmt.Sprintf("Internet search: %s", query)
		s = utils.WrapTextByWords(s, 64)
		return nil, &s

	case globals.ModeSearchProgram:
		systemResults := systemactions.Find(query)
		appResults := apps.FindAppResults(query)
		documentResults := documents.FilterDocumentsByName(query)
		results := combineResults(appResults, systemResults, documentResults)
		if len(results) == 0 && !calculated {
			results = []globals.Resource{{Name: "Search the web for: " + query, WebSearch: query}}
		}
		return withCalculation(results, calculation, calculated), nil

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
	if result, ok := utils.InstantAnswer(expression); ok {
		return result, true
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
	if len(resources) >= maxCombinedResults {
		resources = resources[:maxCombinedResults-1]
	}
	display := strings.ReplaceAll(calculation, "\n", "  /  ")
	result := make([]globals.Resource, 0, len(resources)+1)
	result = append(result, globals.Resource{Name: display, Computed: true})
	return append(result, resources...)
}

func combineResults(appResults, systemResults, documentResults []globals.Resource) []globals.Resource {
	limit := min(len(appResults), maxAppResults)
	combined := append([]globals.Resource(nil), appResults[:limit]...)
	remaining := maxCombinedResults - len(combined)
	if remaining > 0 {
		limit = min(len(systemResults), min(maxCommandResults, remaining))
		combined = append(combined, systemResults[:limit]...)
	}
	remaining = maxCombinedResults - len(combined)
	if remaining > 0 {
		limit = min(len(documentResults), remaining)
		for _, document := range documentResults[:limit] {
			document.Document = true
			combined = append(combined, document)
		}
	}
	return combined
}

// UpdateSearchSetting updates the saved search-string.
func UpdateSearchSetting(s string) {
	globals.SearchString = s
	_ = settings.SetSetting("searchstring", s)
}

func UpdateAliasSetting(value string) {
	globals.AliasString = value
	apps.SetAliases(value)
	_ = settings.SetSetting("aliases", value)
}
