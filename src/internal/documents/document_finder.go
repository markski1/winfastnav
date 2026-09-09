package documents

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	g "winfastnav/internal/globals"
	"winfastnav/internal/recent"
	"winfastnav/internal/utils"
)

var (
	DocumentCache       []g.Resource
	documentCacheMu     sync.RWMutex
	documentChangedMu   sync.RWMutex
	documentChangedHook func()
)

func SetChangedHandler(handler func()) {
	documentChangedMu.Lock()
	documentChangedHook = handler
	documentChangedMu.Unlock()
}

func SetupDocs() {
	log.Print("Indexing documents")
	var documentCache []g.Resource

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("failed to get homedir: %v", err)
		return
	}

	skipIfContains := []string{
		"\\node_modules\\",
		"\\venv\\",
		"\\__pycache__\\",
		"\\sdk-manifests\\",
		"\\sdk\\",
	}

	err = filepath.Walk(homeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if isHiddenDir(info) {
				return filepath.SkipDir
			}
			if utils.ContainsAny(strings.ToLower(path), skipIfContains) {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if normalizeDocumentExtension(ext) == "" {
			return nil
		}

		doc := g.Resource{
			Name:       info.Name(),
			Filepath:   path,
			SearchName: strings.ToLower(info.Name()),
			SearchPath: strings.ToLower(filepath.Dir(path)),
		}
		documentCache = append(documentCache, doc)

		return nil
	})

	if err != nil {
		fmt.Printf("Warning: failed to search path %s: %v\n", homeDir, err)
	}

	documentCacheMu.Lock()
	DocumentCache = documentCache
	documentCacheMu.Unlock()

	log.Print("Documents indexed")
	documentChangedMu.RLock()
	handler := documentChangedHook
	documentChangedMu.RUnlock()
	if handler != nil {
		handler()
	}
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
	recentPaths := recent.Paths()
	collector := newDocumentCollector(30, recentPaths)

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
			collector.add(doc)
		}
	}
	return collector.results()
}

type rankedDocument struct {
	resource g.Resource
	order    int
}

type documentCollector struct {
	limit       int
	recentPaths []string
	recent      []rankedDocument
	ordinary    []g.Resource
}

func newDocumentCollector(limit int, recentPaths []string) documentCollector {
	return documentCollector{
		limit:       limit,
		recentPaths: recentPaths,
		recent:      make([]rankedDocument, 0, len(recentPaths)),
		ordinary:    make([]g.Resource, 0, limit),
	}
}

func (collector *documentCollector) add(resource g.Resource) {
	for order, path := range collector.recentPaths {
		if strings.EqualFold(path, resource.Filepath) {
			collector.recent = append(collector.recent, rankedDocument{resource: resource, order: order})
			return
		}
	}
	if len(collector.ordinary) < collector.limit {
		collector.ordinary = append(collector.ordinary, resource)
	}
}

func (collector *documentCollector) results() []g.Resource {
	sort.Slice(collector.recent, func(i, j int) bool { return collector.recent[i].order < collector.recent[j].order })
	result := make([]g.Resource, 0, collector.limit)
	for _, document := range collector.recent {
		result = append(result, document.resource)
		if len(result) == collector.limit {
			return result
		}
	}
	remaining := collector.limit - len(result)
	if remaining > len(collector.ordinary) {
		remaining = len(collector.ordinary)
	}
	return append(result, collector.ordinary[:remaining]...)
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

func RecentDocuments() []g.Resource {
	documentCacheMu.RLock()
	resources := append([]g.Resource(nil), DocumentCache...)
	documentCacheMu.RUnlock()
	return limitDocuments(recent.Only(resources))
}

func limitDocuments(resources []g.Resource) []g.Resource {
	if len(resources) > 30 {
		return resources[:30]
	}
	return resources
}

func OpenFile(path string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	err := cmd.Start()
	if err == nil {
		recent.Record(path)
	}
	return err
}
