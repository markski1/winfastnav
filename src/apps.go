package main

import (
	"golang.org/x/sys/windows/registry"
	"log"
	"os/exec"
	"sort"
	"strings"
	d "winfastnav/assets"
)

var (
	appDict []d.App
)

func setupApps() {
	log.Printf("Obtaining Windows Applications")
	appDict = getInstalledApps()
	log.Printf("Done")
}

func findAppResults(needle string) []d.App {
	var results []d.App

	needle = strings.ToLower(needle)

	for _, app := range appDict {
		if strings.Contains(strings.ToLower(app.Name), needle) {
			results = append(results, app)
		}
	}

	return results
}

func getInstalledApps() []d.App {
	keys := []registry.Key{
		registry.LOCAL_MACHINE,
		registry.CURRENT_USER,
	}
	basePaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	// We only care about programs
	skipRelease := map[string]struct{}{
		"hotfix":          {},
		"security update": {},
		"service pack":    {},
		"update":          {},
	}

	// We don't care about stuff with these substr either
	skipIfSubstr := []string{
		"speech recognition",
		"redistributable",
		"x64-based systems",
	}

	var apps []d.App

	count := 1

	for _, keyRoot := range keys {
		for _, basePath := range basePaths {
			k, err := registry.OpenKey(keyRoot, basePath, registry.READ)

			if err != nil {
				continue
			}

			names, err := k.ReadSubKeyNames(-1)

			_ = k.Close()

			if err != nil {
				continue
			}

			// Go through each application subkey
			for _, name := range names {
				subKey, err := registry.OpenKey(keyRoot, basePath+`\`+name, registry.READ)
				if err != nil {
					continue
				}

				// Gotta have a name
				displayName, _, err := subKey.GetStringValue("DisplayName")
				if err != nil || strings.TrimSpace(displayName) == "" {
					_ = subKey.Close()
					continue
				}

				// Remove if they contain any substring from skipIfSubstr
				if containsAny(strings.ToLower(displayName), skipIfSubstr) {
					_ = subKey.Close()
				}

				// no system components
				if sysVal, _, err := subKey.GetIntegerValue("SystemComponent"); err == nil && sysVal > 0 {
					_ = subKey.Close()
					continue
				}

				// skip releases in skipRelease
				if rel, _, err := subKey.GetStringValue("ReleaseType"); err == nil {
					if _, bad := skipRelease[strings.ToLower(rel)]; bad {
						_ = subKey.Close()
						continue
					}
				}

				execPath, _, err := subKey.GetStringValue("DisplayIcon")

				// Sometimes there's no exec path, we can do nothing with those!
				if err != nil || len(execPath) < 1 {
					_ = subKey.Close()
					continue
				}

				// Sometimes there's a comma and extra params, clear those out
				apps = append(apps, d.App{Id: count, Name: displayName, ExecPath: cleanExecutablePath(execPath)})
				count++
				_ = subKey.Close()
			}
		}
	}

	// sort by name
	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name)
	})

	return apps
}

func openProgram(execPath string) {
	cmd := exec.Command(execPath)
	_ = cmd.Start()
	hideWindow()
}

func cleanExecutablePath(path string) string {
	if i := strings.Index(path, ","); i != -1 {
		path = path[:i]
	}
	return strings.TrimSpace(path)
}
