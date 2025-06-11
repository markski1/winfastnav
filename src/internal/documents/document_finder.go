package documents

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	g "winfastnav/internal/globals"
	"winfastnav/internal/utils"
)

var (
	DocumentCache []g.Resource
)

func SetupDocs() {
	log.Print("Indexing documents")
	DocumentCache = make([]g.Resource, 0)

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

	relevantExtensions := []string{
		".doc",
		".docx",
		".pdf",
		".rtf",
		".odt",
		".xls",
		".xlsx",
		".ppt",
		".pptx",
	}

	err = filepath.Walk(homeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if isHiddenDir(info) {
				return filepath.SkipDir
			}
			if utils.ContainsAny(path, skipIfContains) {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !utils.ContainsAny(ext, relevantExtensions) {
			return nil
		}

		doc := g.Resource{
			Name:     info.Name(),
			Filepath: path,
		}
		DocumentCache = append(DocumentCache, doc)

		return nil
	})

	if err != nil {
		fmt.Printf("Warning: failed to search path %s: %v\n", homeDir, err)
	}

	log.Print("Documents indexed")
	g.FinishedCachingDocs = true
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
	var filtered []g.Resource
	pattern := strings.ToLower(namePattern)

	for _, doc := range DocumentCache {
		if strings.Contains(strings.ToLower(doc.Name), pattern) {
			filtered = append(filtered, doc)
		}
		if len(filtered) >= 30 {
			break
		}
	}

	return filtered
}

func OpenFile(path string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return cmd.Start()
}
