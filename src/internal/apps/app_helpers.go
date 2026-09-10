package apps

import (
	"encoding/json"
	"log"
	"sort"
	"strings"
	g "winfastnav/internal/globals"
	"winfastnav/internal/settings"
)

func BlockedApplications() []string {
	appListMu.RLock()
	blocklist := append([]string(nil), g.ExecBlocklist...)
	appListMu.RUnlock()
	sort.Strings(blocklist)
	return blocklist
}

func UnblockApplication(path string) error {
	appListMu.Lock()
	blocklist := g.ExecBlocklist[:0]
	for _, blockedPath := range g.ExecBlocklist {
		if !strings.EqualFold(blockedPath, path) {
			blocklist = append(blocklist, blockedPath)
		}
	}
	g.ExecBlocklist = blocklist
	blocklist = append([]string(nil), g.ExecBlocklist...)
	appListMu.Unlock()

	if err := saveBlocklist(blocklist); err != nil {
		return err
	}
	go SetupApps()
	return nil
}

func saveBlocklist(blocklist []string) error {
	jsonData, err := json.Marshal(blocklist)
	if err != nil {
		return err
	}
	return settings.SetSetting("blocklist", string(jsonData))
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

	if err := saveBlocklist(blocklist); err != nil {
		log.Printf("Error saving settings: %v", err)
	}
}
