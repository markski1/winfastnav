package main

import (
	"golang.org/x/sys/windows/registry"
	"log"
	"strings"
)

var (
	appDict map[string]string
)

func setupApps() {
	appDict = getInstalledApps()
}

func findAppResults(needle string) (results map[string]string) {
	results = make(map[string]string)

	for keyName, valueName := range appDict {
		if strings.Contains(strings.ToLower(keyName), strings.ToLower(needle)) {
			results[keyName] = valueName
		}
	}

	return
}

func getInstalledApps() map[string]string {
	keys := []registry.Key{
		registry.LOCAL_MACHINE,
		registry.CURRENT_USER,
	}
	basePaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall`, // for 32-bit apps on 64-bit Windows
	}

	apps := make(map[string]string)

	for _, keyRoot := range keys {
		for _, basePath := range basePaths {
			k, err := registry.OpenKey(keyRoot, basePath, registry.READ)

			if err != nil {
				continue
			}

			names, err := k.ReadSubKeyNames(-1)

			_ = k.Close()

			if err != nil {
				log.Print(err)
				continue
			}

			for _, name := range names {
				subKey, err := registry.OpenKey(keyRoot, basePath+`\`+name, registry.READ)
				if err != nil {
					continue
				}

				displayName, _, err := subKey.GetStringValue("DisplayName")
				if err != nil || strings.TrimSpace(displayName) == "" {
					_ = subKey.Close()
					continue
				}

				displayIcon, _, err := subKey.GetStringValue("DisplayIcon")
				// DisplayIcon may contain ",0" at the end or other parameters, so clean it
				if err == nil {
					displayIcon = cleanExecutablePath(displayIcon)
					apps[displayName] = displayIcon
				} else {
					apps[displayName] = ""
				}

				_ = subKey.Close()
			}
		}
	}
	return apps
}

func cleanExecutablePath(path string) string {
	if i := strings.Index(path, ","); i != -1 {
		path = path[:i]
	}
	return strings.TrimSpace(path)
}
