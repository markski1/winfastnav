package documents

import (
	"testing"

	g "winfastnav/internal/globals"
)

func TestDocumentSearchUsesFilenameSubstringOnly(t *testing.T) {
	documentCacheMu.Lock()
	previous := DocumentCache
	DocumentCache = []g.Resource{
		{Name: "Visual Studio Code.pdf", Filepath: `C:\docs\code.pdf`},
		{Name: "Project notes.pdf", Filepath: `C:\docs\notes.pdf`},
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
