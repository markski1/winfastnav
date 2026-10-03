package documents

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
	g "winfastnav/internal/globals"
	"winfastnav/internal/settings"
	"winfastnav/internal/storage"
)

const documentCacheVersion = 1

type IndexStatus string

const (
	IndexStatusNotReady IndexStatus = "not_ready"
	IndexStatusIndexing IndexStatus = "indexing"
	IndexStatusReady    IndexStatus = "ready"
	IndexStatusStale    IndexStatus = "stale"
	IndexStatusError    IndexStatus = "error"
)

type IndexSnapshot struct {
	Status      IndexStatus
	ItemCount   int
	RootCount   int
	LastIndexed time.Time
	Error       string
}

type IndexConfig struct {
	Roots      []string `json:"roots"`
	Exclusions []string `json:"exclusions"`
}

type documentCacheFile struct {
	Version   int          `json:"version"`
	Config    IndexConfig  `json:"config"`
	IndexedAt time.Time    `json:"indexed_at"`
	Documents []g.Resource `json:"documents"`
}

var (
	DocumentCache       []g.Resource
	documentCacheMu     sync.RWMutex
	documentChangedMu   sync.RWMutex
	documentChangedHook func()
	indexStatusMu       sync.RWMutex
	indexStatusHook     func(IndexSnapshot)
	indexState          = IndexSnapshot{Status: IndexStatusNotReady}
	indexConfigMu       sync.RWMutex
	indexConfig         IndexConfig
	indexConfigReady    bool
	indexRunMu          sync.Mutex
	indexRunning        bool
	indexPending        bool
)

func SetChangedHandler(handler func()) {
	documentChangedMu.Lock()
	documentChangedHook = handler
	documentChangedMu.Unlock()
}

func SetStatusChangedHandler(handler func(IndexSnapshot)) {
	indexStatusMu.Lock()
	indexStatusHook = handler
	indexStatusMu.Unlock()
}

func Status() IndexSnapshot {
	indexStatusMu.RLock()
	defer indexStatusMu.RUnlock()
	return indexState
}

func Config() IndexConfig {
	indexConfigMu.Lock()
	if !indexConfigReady {
		indexConfig = defaultIndexConfig()
		indexConfigReady = true
	}
	config := copyIndexConfig(indexConfig)
	indexConfigMu.Unlock()
	return config
}

func LoadConfigFromSettings() {
	rootsValue := settings.GetSetting("indexroots")
	exclusionsValue := settings.GetSetting("indexexclusions")
	config := IndexConfig{
		Roots:      normalizeRoots(ParseIndexList(rootsValue)),
		Exclusions: normalizeExclusions(ParseIndexList(exclusionsValue)),
	}
	setConfig(config)
}

func ApplyConfig(config IndexConfig) (bool, error) {
	config = normalizeIndexConfig(config)
	if err := settings.SetSettings(settings.Settings{
		"indexroots":      strings.Join(config.Roots, ";"),
		"indexexclusions": strings.Join(config.Exclusions, ";"),
	}); err != nil {
		return false, err
	}
	changed := setConfig(config)
	if changed {
		state := Status()
		setIndexState(IndexStatusStale, state.ItemCount, len(config.Roots), state.LastIndexed, nil)
	}
	return changed, nil
}

func ParseIndexList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == '\r' || r == '\n'
	})
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			result = append(result, field)
		}
	}
	return result
}

func SetupDocs() {
	indexRunMu.Lock()
	indexPending = true
	if indexRunning {
		indexRunMu.Unlock()
		return
	}
	indexRunning = true
	indexRunMu.Unlock()

	for {
		indexRunMu.Lock()
		if !indexPending {
			indexRunning = false
			indexRunMu.Unlock()
			return
		}
		indexPending = false
		indexRunMu.Unlock()
		indexDocuments()
	}
}

func indexDocuments() {
	config := Config()
	state := Status()
	setIndexState(IndexStatusIndexing, state.ItemCount, len(config.Roots), state.LastIndexed, nil)

	if cached, ok := loadDocumentCache(config); ok {
		documentCacheMu.Lock()
		DocumentCache = cached.Documents
		documentCacheMu.Unlock()
		setIndexState(IndexStatusIndexing, len(cached.Documents), len(config.Roots), cached.IndexedAt, nil)
		notifyDocumentChanged()
	}

	log.Printf("Indexing documents in %d root(s)", len(config.Roots))
	var documentCache []g.Resource
	var indexErrors []string
	for _, root := range config.Roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				indexErrors = append(indexErrors, path+": "+err.Error())
				return nil
			}
			if info == nil {
				return nil
			}

			if info.IsDir() {
				if shouldSkipDirectory(path, info, config) {
					return filepath.SkipDir
				}
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if normalizeDocumentExtension(ext) == "" {
				return nil
			}

			documentCache = append(documentCache, newDocumentResource(path, info.Name()))
			return nil
		})
		if err != nil {
			indexErrors = append(indexErrors, root+": "+err.Error())
		}
	}
	sort.Slice(documentCache, func(i, j int) bool {
		return strings.ToLower(documentCache[i].Filepath) < strings.ToLower(documentCache[j].Filepath)
	})

	documentCacheMu.Lock()
	DocumentCache = documentCache
	documentCacheMu.Unlock()

	indexedAt := time.Now()
	if err := saveDocumentCache(documentCacheFile{Version: documentCacheVersion, Config: config, IndexedAt: indexedAt, Documents: documentCache}); err != nil {
		indexErrors = append(indexErrors, "cache: "+err.Error())
	}
	if !sameIndexConfig(config, Config()) {
		setIndexState(IndexStatusStale, len(documentCache), len(config.Roots), indexedAt, nil)
		indexRunMu.Lock()
		indexPending = true
		indexRunMu.Unlock()
		return
	}

	status := IndexStatusReady
	var statusErr error
	if len(indexErrors) > 0 {
		status = IndexStatusError
		statusErr = errors.New(strings.Join(indexErrors, "\n"))
	}
	setIndexState(status, len(documentCache), len(config.Roots), indexedAt, statusErr)
	notifyDocumentChanged()
	log.Printf("Documents indexed: %d", len(documentCache))
}

func defaultIndexConfig() IndexConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}
	return IndexConfig{
		Roots:      normalizeRoots([]string{homeDir}),
		Exclusions: normalizeExclusions([]string{"node_modules", "venv", "__pycache__", "sdk-manifests", "sdk"}),
	}
}

func normalizeIndexConfig(config IndexConfig) IndexConfig {
	config.Roots = normalizeRoots(config.Roots)
	config.Exclusions = normalizeExclusions(config.Exclusions)
	defaults := defaultIndexConfig()
	if len(config.Roots) == 0 {
		config.Roots = defaults.Roots
	}
	if len(config.Exclusions) == 0 {
		config.Exclusions = defaults.Exclusions
	}
	return config
}

func normalizeRoots(values []string) []string {
	var result []string
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(expandIndexPath(value)), `"`)
		if value == "" {
			continue
		}
		if absolute, err := filepath.Abs(value); err == nil {
			value = absolute
		}
		value = filepath.Clean(value)
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeExclusions(values []string) []string {
	var result []string
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.Trim(strings.TrimSpace(expandIndexPath(value)), `"`)
		if value == "" {
			continue
		}
		if filepath.IsAbs(value) {
			value = filepath.Clean(value)
		} else {
			value = strings.Trim(value, `\\/`)
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func expandIndexPath(value string) string {
	value = os.ExpandEnv(value)
	for start := 0; start < len(value); {
		open := strings.IndexByte(value[start:], '%')
		if open < 0 {
			break
		}
		open += start
		close := strings.IndexByte(value[open+1:], '%')
		if close < 0 {
			break
		}
		close += open + 1
		name := value[open+1 : close]
		if name == "" {
			start = close + 1
			continue
		}
		replacement, ok := os.LookupEnv(name)
		if !ok {
			start = close + 1
			continue
		}
		value = value[:open] + replacement + value[close+1:]
		start = open + len(replacement)
	}
	return value
}

func setConfig(config IndexConfig) bool {
	config = normalizeIndexConfig(config)
	indexConfigMu.Lock()
	changed := !sameIndexConfig(indexConfig, config) || !indexConfigReady
	indexConfig = copyIndexConfig(config)
	indexConfigReady = true
	indexConfigMu.Unlock()
	return changed
}

func copyIndexConfig(config IndexConfig) IndexConfig {
	return IndexConfig{
		Roots:      append([]string(nil), config.Roots...),
		Exclusions: append([]string(nil), config.Exclusions...),
	}
}

func sameIndexConfig(left, right IndexConfig) bool {
	return slices.EqualFunc(left.Roots, right.Roots, strings.EqualFold) && slices.EqualFunc(left.Exclusions, right.Exclusions, strings.EqualFold)
}

func shouldSkipDirectory(path string, info os.FileInfo, config IndexConfig) bool {
	if isHiddenDir(info) {
		return true
	}
	cleanPath := strings.ToLower(filepath.Clean(path))
	for _, exclusion := range config.Exclusions {
		cleanExclusion := strings.ToLower(filepath.Clean(exclusion))
		if filepath.IsAbs(exclusion) {
			if cleanPath == cleanExclusion || strings.HasPrefix(cleanPath, cleanExclusion+string(filepath.Separator)) {
				return true
			}
			continue
		}
		if strings.EqualFold(info.Name(), exclusion) {
			return true
		}
	}
	return false
}

func newDocumentResource(path, name string) g.Resource {
	return g.Resource{
		Name:       name,
		Filepath:   path,
		SearchName: strings.ToLower(name),
		SearchPath: strings.ToLower(filepath.Dir(path)),
	}
}

func setIndexState(status IndexStatus, itemCount, rootCount int, indexedAt time.Time, indexErr error) {
	snapshot := IndexSnapshot{Status: status, ItemCount: itemCount, RootCount: rootCount, LastIndexed: indexedAt}
	if indexErr != nil {
		snapshot.Error = indexErr.Error()
	}
	indexStatusMu.Lock()
	indexState = snapshot
	handler := indexStatusHook
	indexStatusMu.Unlock()
	if handler != nil {
		handler(snapshot)
	}
}

func notifyDocumentChanged() {
	documentChangedMu.RLock()
	handler := documentChangedHook
	documentChangedMu.RUnlock()
	if handler != nil {
		handler()
	}
}

func documentCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "winfastnav", "documents-v1.json"), nil
}

func loadDocumentCache(config IndexConfig) (documentCacheFile, bool) {
	path, err := documentCachePath()
	if err != nil {
		return documentCacheFile{}, false
	}
	file, err := os.Open(path)
	if err != nil {
		return documentCacheFile{}, false
	}
	defer file.Close()

	var cached documentCacheFile
	if json.NewDecoder(file).Decode(&cached) != nil || cached.Version != documentCacheVersion || !sameIndexConfig(cached.Config, config) {
		return documentCacheFile{}, false
	}
	for index := range cached.Documents {
		cached.Documents[index] = newDocumentResource(cached.Documents[index].Filepath, cached.Documents[index].Name)
	}
	return cached, true
}

func saveDocumentCache(cached documentCacheFile) error {
	path, err := documentCachePath()
	if err != nil {
		return err
	}
	return storage.WriteJSON(path, cached, "")
}

func isHiddenDir(info os.FileInfo) bool {
	if strings.HasPrefix(info.Name(), ".") {
		return true
	}

	if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return stat.FileAttributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0
	}

	return false
}

func FilterDocumentsByName(namePattern string) []g.Resource {
	query := parseDocumentQuery(namePattern)
	results := make([]g.Resource, 0, 30)

	documentCacheMu.RLock()
	defer documentCacheMu.RUnlock()
	for _, doc := range DocumentCache {
		name := doc.SearchName
		if name == "" {
			name = strings.ToLower(doc.Name)
		}
		parent := doc.SearchPath
		if parent == "" {
			parent = strings.ToLower(filepath.Dir(doc.Filepath))
		}
		if query.extension != "" && !strings.EqualFold(filepath.Ext(doc.Filepath), query.extension) {
			continue
		}
		if query.folder != "" && !strings.Contains(parent, query.folder) {
			continue
		}
		matched := true
		for _, term := range query.terms {
			if !strings.Contains(name, term) && !strings.Contains(parent, term) {
				matched = false
				break
			}
		}
		if matched {
			results = append(results, doc)
			if len(results) == cap(results) {
				break
			}
		}
	}
	return results
}

type documentQuery struct {
	terms     []string
	folder    string
	extension string
}

func parseDocumentQuery(value string) documentQuery {
	var query documentQuery
	for _, field := range strings.Fields(strings.ToLower(value)) {
		extension := normalizeDocumentExtension(field)
		switch {
		case strings.HasPrefix(field, "folder:"):
			query.folder = strings.TrimPrefix(field, "folder:")
		case strings.HasPrefix(field, "type:"):
			query.extension = normalizeDocumentExtension(strings.TrimPrefix(field, "type:"))
		case extension != "":
			query.extension = extension
		default:
			query.terms = append(query.terms, field)
		}
	}
	return query
}

func normalizeDocumentExtension(value string) string {
	value = strings.TrimPrefix(value, ".")
	switch value {
	case "doc", "docx", "pdf", "rtf", "odt", "xls", "xlsx", "ppt", "pptx":
		return "." + value
	default:
		return ""
	}
}

func OpenFile(path string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return cmd.Start()
}
