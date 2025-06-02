package main

import (
	"log"
	"os/exec"
	"strings"
	d "winfastnav/assets"
)

func setupApps() {
	log.Printf("Obtaining Windows Applications")
	d.AppList = d.GetInstalledApps()
	log.Printf("Done")
}

func findAppResults(needle string) []d.App {
	var results []d.App

	needle = strings.ToLower(needle)

	for _, app := range d.AppList {
		if strings.Contains(strings.ToLower(app.Name), needle) || strings.Contains(strings.ToLower(app.ExecPath), needle) {
			results = append(results, app)
			if len(results) >= 20 {
				break
			}
		}
	}

	return results
}

func openProgram(execPath string) {
	cmd := exec.Command(execPath)
	_ = cmd.Start()
	hideWindow()
}
