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

type CatalogPhase string

const (
	CatalogPhaseNotReady CatalogPhase = "not_ready"
	CatalogPhaseIndexing CatalogPhase = "indexing"
	CatalogPhaseReady    CatalogPhase = "ready"
	CatalogPhaseError    CatalogPhase = "error"
)

type CatalogSnapshot struct {
	Phase       CatalogPhase
	ItemCount   int
	LastIndexed time.Time
	Error       string
}

type catalogFile struct {
	Version     int          `json:"version"`
	Fingerprint uint64       `json:"fingerprint"`
	IndexedAt   time.Time    `json:"indexed_at"`
	Apps        []g.Resource `json:"apps"`
}

var (
	catalogMu          sync.Mutex
	refreshMu          sync.Mutex
	catalogFingerprint uint64
	catalogChanged     func()
	catalogStatusMu    sync.RWMutex
	catalogStatusHook  func(CatalogSnapshot)
	catalogState       = CatalogSnapshot{Phase: CatalogPhaseNotReady}
)

func SetCatalogChangedHandler(handler func()) {
	catalogMu.Lock()
	catalogChanged = handler
	catalogMu.Unlock()
}

func SetStatusChangedHandler(handler func(CatalogSnapshot)) {
	catalogStatusMu.Lock()
	catalogStatusHook = handler
	catalogStatusMu.Unlock()
}

func Status() CatalogSnapshot {
	catalogStatusMu.RLock()
	defer catalogStatusMu.RUnlock()
	return catalogState
}

func LoadCatalog() {
	setCatalogStatus(CatalogPhaseIndexing, catalogItemCount(), time.Time{}, nil)
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
	indexedAt := cached.IndexedAt
	if indexedAt.IsZero() {
		indexedAt = time.Now()
	}
	setCatalogStatus(CatalogPhaseReady, len(apps), indexedAt, nil)
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
	setCatalogStatus(CatalogPhaseIndexing, catalogItemCount(), time.Time{}, nil)
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
	indexedAt := time.Now()
	if err := saveCatalog(catalogFile{Version: catalogVersion, Fingerprint: fingerprint, IndexedAt: indexedAt, Apps: apps}); err != nil {
		log.Printf("failed to save app catalog: %v", err)
		setCatalogStatus(CatalogPhaseError, len(apps), indexedAt, err)
	} else {
		setCatalogStatus(CatalogPhaseReady, len(apps), indexedAt, nil)
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
	setCatalogStatus(CatalogPhaseReady, len(apps), time.Now(), nil)
}

func catalogItemCount() int {
	appListMu.RLock()
	defer appListMu.RUnlock()
	return len(g.AppList)
}

func setCatalogStatus(phase CatalogPhase, itemCount int, indexedAt time.Time, statusErr error) {
	snapshot := CatalogSnapshot{Phase: phase, ItemCount: itemCount, LastIndexed: indexedAt}
	if statusErr != nil {
		snapshot.Error = statusErr.Error()
	}
	catalogStatusMu.Lock()
	catalogState = snapshot
	handler := catalogStatusHook
	catalogStatusMu.Unlock()
	if handler != nil {
		handler(snapshot)
	}
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
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, ".apps-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := json.NewEncoder(temporary).Encode(catalog); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
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
