package apps

import (
	"log"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	g "winfastnav/internal/globals"
	"winfastnav/internal/recent"
)

var appListMu sync.RWMutex

func SetupApps() {
	log.Printf("Indexing Windows apps")
	appList := GetInstalledApps()
	appListMu.Lock()
	g.AppList = appList
	appListMu.Unlock()
	log.Printf("Windows apps indexed")
}

func FindAppResults(needle string) []g.Resource {
	var results []g.Resource

	needle = strings.ToLower(needle)

	appListMu.RLock()
	defer appListMu.RUnlock()
	for _, app := range g.AppList {
		if strings.Contains(strings.ToLower(app.Name), needle) || strings.Contains(strings.ToLower(app.Filepath), needle) {
			results = append(results, app)
		}
	}
	return limitResults(recent.Rank(results))
}

func RecentApplications() []g.Resource {
	appListMu.RLock()
	resources := append([]g.Resource(nil), g.AppList...)
	appListMu.RUnlock()
	return limitResults(recent.Only(resources))
}

func limitResults(resources []g.Resource) []g.Resource {
	if len(resources) > 30 {
		return resources[:30]
	}
	return resources
}

func OpenProgram(execPath string) error {
	cmd := exec.Command(execPath)
	if isAppsFolderPath(execPath) {
		cmd = exec.Command("explorer.exe", execPath)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	err := cmd.Start()
	if err == nil {
		recent.Record(execPath)
	}
	return err
}
