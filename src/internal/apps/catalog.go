package apps

import (
	"encoding/json"
	"hash/fnv"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	g "winfastnav/internal/globals"
)

const (
	catalogVersion  = 2
	catalogFilename = "apps-v2.json"
	catalogPoll     = 2 * time.Minute
)

type catalogFile struct {
	Version     int          `json:"version"`
	Fingerprint uint64       `json:"fingerprint"`
	Apps        []g.Resource `json:"apps"`
}

var (
	catalogMu          sync.Mutex
	refreshMu          sync.Mutex
	catalogFingerprint uint64
	catalogChanged     func()
)

func SetCatalogChangedHandler(handler func()) {
	catalogMu.Lock()
	catalogChanged = handler
	catalogMu.Unlock()
}

func LoadCatalog() {
	path, err := catalogPath()
	if err != nil {
		seedCatalog()
		return
	}
	file, err := os.Open(path)
	if err != nil {
		seedCatalog()
		return
	}
	defer file.Close()

	var cached catalogFile
	if json.NewDecoder(file).Decode(&cached) != nil || cached.Version != catalogVersion || len(cached.Apps) == 0 {
		seedCatalog()
		return
	}
	apps := filterCachedApps(cached.Apps)
	prepareResources(apps)
	appListMu.Lock()
	g.AppList = apps
	appListMu.Unlock()
	catalogMu.Lock()
	catalogFingerprint = cached.Fingerprint
	catalogMu.Unlock()
}

func MonitorCatalog() {
	refreshCatalogIfChanged()
	ticker := time.NewTicker(catalogPoll)
	defer ticker.Stop()
	for range ticker.C {
		refreshCatalogIfChanged()
	}
}

func refreshCatalogIfChanged() {
	fingerprint := applicationSourcesFingerprint()
	catalogMu.Lock()
	changed := fingerprint == 0 || fingerprint != catalogFingerprint
	catalogMu.Unlock()
	if changed {
		refreshCatalog(fingerprint)
	}
}

func refreshCatalog(fingerprint uint64) {
	refreshMu.Lock()
	defer refreshMu.Unlock()
	log.Printf("Indexing Windows apps")
	apps := GetInstalledApps()
	prepareResources(apps)

	appListMu.Lock()
	changed := !sameResources(g.AppList, apps)
	g.AppList = apps
	appListMu.Unlock()

	catalogMu.Lock()
	catalogFingerprint = fingerprint
	catalogMu.Unlock()
	if err := saveCatalog(catalogFile{Version: catalogVersion, Fingerprint: fingerprint, Apps: apps}); err != nil {
		log.Printf("failed to save app catalog: %v", err)
	}
	if changed {
		catalogMu.Lock()
		handler := catalogChanged
		catalogMu.Unlock()
		if handler != nil {
			handler()
		}
	}
	log.Printf("Windows apps indexed")
}

func seedCatalog() {
	apps := []g.Resource{calculatorResource()}
	prepareResources(apps)
	appListMu.Lock()
	g.AppList = apps
	appListMu.Unlock()
}

func filterCachedApps(apps []g.Resource) []g.Resource {
	filtered := apps[:0]
	for _, app := range apps {
		if !containsAnyFold(strings.ToLower(app.Filepath), g.ExecBlocklist) {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

func saveCatalog(catalog catalogFile) error {
	path, err := catalogPath()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	err = encoder.Encode(catalog)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func catalogPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "winfastnav", catalogFilename), nil
}

func applicationSourcesFingerprint() uint64 {
	hash := fnv.New64a()
	for _, base := range startMenuDirectories() {
		_ = filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".lnk") {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return nil
			}
			_, _ = hash.Write([]byte(strings.ToLower(path)))
			_, _ = hash.Write([]byte(strconv.FormatInt(info.ModTime().UnixNano(), 10)))
			_, _ = hash.Write([]byte(strconv.FormatInt(info.Size(), 10)))
			return nil
		})
	}
	packages := filepath.Join(os.Getenv("LOCALAPPDATA"), "Packages")
	if entries, err := os.ReadDir(packages); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				_, _ = hash.Write([]byte(strings.ToLower(entry.Name())))
			}
		}
	}

	return hash.Sum64()
}

func sameResources(left, right []g.Resource) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func prepareResources(resources []g.Resource) {
	for index := range resources {
		resources[index].SearchName = strings.ToLower(strings.TrimSpace(resources[index].Name))
		resources[index].SearchPath = strings.ToLower(resources[index].Filepath)
	}
}
