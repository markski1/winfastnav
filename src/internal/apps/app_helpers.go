package apps

import (
	"encoding/json"
	"log"
	g "winfastnav/internal/globals"
	"winfastnav/internal/settings"
)

func UnblockAllApplications() {
	appListMu.Lock()
	g.ExecBlocklist = []string{}
	appListMu.Unlock()

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

	go SetupApps()
}

func BlockApplication(application g.Resource) {
	appListMu.Lock()
	defer appListMu.Unlock()
	for i, app := range g.AppList {
		if app == application {
			g.AppList = append(g.AppList[:i], g.AppList[i+1:]...)
			break
		}
	}

	g.ExecBlocklist = append(g.ExecBlocklist, application.Filepath)
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
