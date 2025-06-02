package assets

import "strings"
import (
	"encoding/json"
	"log"
	"strings"
)

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
func BlockApplication(application App) {
	for i, app := range AppList {
		if app == application {
			AppList = append(AppList[:i], AppList[i+1:]...)
			break
		}
	}

	ExecBlocklist = append(ExecBlocklist, application.ExecPath)
	jsonData, err := json.Marshal(ExecBlocklist)
	if err != nil {
		log.Printf("Error encoding list to JSON: %v", err)
		return
	}
	err = setSetting("blocklist", string(jsonData))
	if err != nil {
		log.Printf("Error saving settings: %v", err)
		return
	}
}
