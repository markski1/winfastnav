package documents

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	g "winfastnav/internal/globals"
)

func TestDocumentSearchUsesFilenameSubstringOnly(t *testing.T) {
	documentCacheMu.Lock()
	previous := DocumentCache
	DocumentCache = []g.Resource{
		{Name: "Visual Studio Code.pdf", Filepath: `C:\docs\work\code.pdf`},
		{Name: "Project notes.docx", Filepath: `C:\docs\personal\notes.docx`},
	}
	documentCacheMu.Unlock()
	t.Cleanup(func() {
		documentCacheMu.Lock()
		DocumentCache = previous
		documentCacheMu.Unlock()
	})

	if results := FilterDocumentsByName("vsc"); len(results) != 0 {
		t.Fatalf("fuzzy query returned documents: %#v", results)
	}
	results := FilterDocumentsByName("studio code")
	if len(results) != 1 || results[0].Name != "Visual Studio Code.pdf" {
		t.Fatalf("substring query returned %#v", results)
	}
}

func TestDocumentSearchSupportsTypeAndFolderFilters(t *testing.T) {
	documentCacheMu.Lock()
	previous := DocumentCache
	DocumentCache = []g.Resource{
		{Name: "Quarterly report.pdf", Filepath: `C:\docs\work\report.pdf`},
		{Name: "Quarterly report.docx", Filepath: `C:\docs\personal\report.docx`},
	}
	documentCacheMu.Unlock()
	t.Cleanup(func() {
		documentCacheMu.Lock()
		DocumentCache = previous
		documentCacheMu.Unlock()
	})

	results := FilterDocumentsByName("pdf folder:work report")
	if len(results) != 1 || filepath.Ext(results[0].Filepath) != ".pdf" {
		t.Fatalf("filtered documents = %#v", results)
	}
}

func TestDocumentSearchMatchesParentFolder(t *testing.T) {
	documentCacheMu.Lock()
	previous := DocumentCache
	DocumentCache = []g.Resource{{Name: "notes.docx", Filepath: `C:\clients\acme\notes.docx`}}
	documentCacheMu.Unlock()
	t.Cleanup(func() {
		documentCacheMu.Lock()
		DocumentCache = previous
		documentCacheMu.Unlock()
	})

	results := FilterDocumentsByName("acme")
	if len(results) != 1 {
		t.Fatalf("parent-folder search returned %#v", results)
	}
}

func TestDocumentExtensionMustMatchExactly(t *testing.T) {
	if extension := normalizeDocumentExtension(".docx.bak"); extension != "" {
		t.Fatalf("invalid extension normalized to %q", extension)
	}
}

func TestParseIndexList(t *testing.T) {
	values := ParseIndexList(" C:\\Docs;D:\\Projects\r\nE:\\Notes ")
	if len(values) != 3 || values[0] != `C:\Docs` || values[1] != `D:\Projects` || values[2] != `E:\Notes` {
		t.Fatalf("parsed index list = %#v", values)
	}
}

func TestNormalizeRootsExpandsWindowsEnvironmentVariables(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WINFASTNAV_TEST_ROOT", root)

	config := normalizeIndexConfig(IndexConfig{Roots: []string{"%WINFASTNAV_TEST_ROOT%\\Documents"}})
	want := filepath.Join(root, "Documents")
	if len(config.Roots) != 1 || !strings.EqualFold(config.Roots[0], want) {
		t.Fatalf("normalized roots = %#v, want %q", config.Roots, want)
	}
}

func TestDocumentCollectorPromotesRecentMatchBeyondLimit(t *testing.T) {
	collector := newDocumentCollector(2, []string{`C:\docs\recent.pdf`})
	collector.add(g.Resource{Name: "First", Filepath: `C:\docs\first.pdf`})
	collector.add(g.Resource{Name: "Second", Filepath: `C:\docs\second.pdf`})
	collector.add(g.Resource{Name: "Recent", Filepath: `C:\docs\recent.pdf`})
	results := collector.results()
	if len(results) != 2 || results[0].Name != "Recent" || results[1].Name != "First" {
		t.Fatalf("collected documents = %#v", results)
	}
}

func BenchmarkFilterTenThousandDocuments(b *testing.B) {
	documents := make([]g.Resource, 10000)
	for index := range documents {
		name := fmt.Sprintf("report-%05d.pdf", index)
		documents[index] = g.Resource{
			Name:       name,
			Filepath:   `C:\docs\work\` + name,
			SearchName: name,
			SearchPath: `c:\docs\work`,
		}
	}
	documentCacheMu.Lock()
	previous := DocumentCache
	DocumentCache = documents
	documentCacheMu.Unlock()
	b.Cleanup(func() {
		documentCacheMu.Lock()
		DocumentCache = previous
		documentCacheMu.Unlock()
	})

	b.ResetTimer()
	for b.Loop() {
		_ = FilterDocumentsByName("pdf folder:work report")
	}
}
