package assets

import (
	"encoding/json"
	"log"
	"os/exec"
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

func UnblockAllApplications() {
	ExecBlocklist = []string{}

	jsonData, err := json.Marshal(ExecBlocklist)
	if err != nil {
		log.Printf("Error encoding list to JSON: %v", err)
		return
	}
	err = SetSetting("blocklist", string(jsonData))
	if err != nil {
		log.Printf("Error saving settings: %v", err)
		return
	}

	AppList = GetInstalledApps()
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
	err = SetSetting("blocklist", string(jsonData))
	if err != nil {
		log.Printf("Error saving settings: %v", err)
		return
	}
}

func OpenURI(uri string) {
	cmd := exec.Command("cmd", "/c", "start", uri)
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to open URI: %v", err)
	}
}
