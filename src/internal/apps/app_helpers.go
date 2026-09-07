package apps

import (
	"encoding/json"
	"log"
	g "winfastnav/internal/globals"
	"winfastnav/internal/settings"
)

func UnblockAllApplications() {
	appListMu.Lock()
	g.ExecBlocklist = nil
	appListMu.Unlock()

	jsonData, err := json.Marshal([]string{})
	if err != nil {
		log.Printf("Error encoding list to JSON: %v", err)
		return
	}
	err = settings.SetSetting("blocklist", string(jsonData))
	if err != nil {
		log.Printf("Error saving settings: %v", err)
		return
	}

	go SetupApps()
}

func BlockApplication(application g.Resource) {
	appListMu.Lock()
	for i, app := range g.AppList {
		if app == application {
			g.AppList = append(g.AppList[:i], g.AppList[i+1:]...)
			break
		}
	}

	g.ExecBlocklist = append(g.ExecBlocklist, application.Filepath)
	blocklist := append([]string(nil), g.ExecBlocklist...)
	appListMu.Unlock()

	jsonData, err := json.Marshal(blocklist)
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
