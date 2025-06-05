package core

import (
	"fmt"
	"strings"

	"winfastnav/internal/apps"
	"winfastnav/internal/globals"
	"winfastnav/internal/settings"
	"winfastnav/internal/utils"
)

func Search(query string) (retApps []globals.App, resultStr *string) {
	if len(query) == 0 {
		return nil, nil
	}

	// internet search
	if query[0] == '@' {
		s := fmt.Sprintf("Internet search: %s", query[1:])
		return nil, &s
	}

	// math evaluation
	if utils.IsMath(query) {
		expr := strings.ReplaceAll(query, " ", "")
		if val, err := utils.EvalMath(expr); err == nil {
			return nil, &val
		}
	}

	// fallback to app search
	return apps.FindAppResults(query), nil
}

// UpdateSearchSetting updates the saved search-string.
func UpdateSearchSetting(s string) {
	globals.SearchString = s
	_ = settings.SetSetting("searchstring", s)
}
