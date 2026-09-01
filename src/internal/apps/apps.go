package apps

import (
	"errors"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sys/windows"
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

	appListMu.RLock()
	results = append(results, g.AppList...)
	appListMu.RUnlock()
	return limitResults(recent.MatchAndRank(results, needle))
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

func RunProgramElevated(execPath string) error {
	if !filepath.IsAbs(execPath) || !strings.EqualFold(filepath.Ext(execPath), ".exe") {
		return errors.New("selected item cannot be run as administrator")
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	path, err := windows.UTF16PtrFromString(execPath)
	if err != nil {
		return err
	}
	if err = windows.ShellExecute(0, verb, path, nil, nil, 1); err == nil {
		recent.Record(execPath)
	}
	return err
}
