package apps

import (
	"log"
	"os/exec"
	"strings"
	"syscall"
	g "winfastnav/internal/globals"
)

func SetupApps() {
	log.Printf("Obtaining Windows Applications")
	g.AppList = GetInstalledApps()
	log.Printf("Done")
}

func FindAppResults(needle string) []g.App {
	var results []g.App

	needle = strings.ToLower(needle)

	for _, app := range g.AppList {
		if strings.Contains(strings.ToLower(app.Name), needle) || strings.Contains(strings.ToLower(app.ExecPath), needle) {
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
