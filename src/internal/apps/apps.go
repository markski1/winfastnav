package apps

import (
	"log"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	g "winfastnav/internal/globals"
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
			if len(results) >= 30 {
				break
			}
		}
	}

	return results
}

func OpenProgram(execPath string) error {
	cmd := exec.Command(execPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return cmd.Start()
}
