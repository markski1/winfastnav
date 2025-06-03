package apps

import (
	"encoding/json"
	"log"
	"strings"
	g "winfastnav/internal/globals"
	"winfastnav/internal/settings"
)

func ContainsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func UnblockAllApplications() {
	g.ExecBlocklist = []string{}

	jsonData, err := json.Marshal(g.ExecBlocklist)
	if err != nil {
		log.Printf("Error encoding list to JSON: %v", err)
		return
	}
	err = settings.SetSetting("blocklist", string(jsonData))
	if err != nil {
		log.Printf("Error saving settings: %v", err)
		return
	}

	g.AppList = GetInstalledApps()
}

func BlockApplication(application g.App) {
	for i, app := range g.AppList {
		if app == application {
			g.AppList = append(g.AppList[:i], g.AppList[i+1:]...)
			break
		}
	}

	g.ExecBlocklist = append(g.ExecBlocklist, application.ExecPath)
	jsonData, err := json.Marshal(g.ExecBlocklist)
	if err != nil {
		log.Printf("Error encoding list to JSON: %v", err)
		return
	}
	err = settings.SetSetting("blocklist", string(jsonData))
	if err != nil {
		log.Printf("Error saving settings: %v", err)
		return
	}
}
