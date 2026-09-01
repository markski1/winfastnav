package recent

import (
	"testing"

	"winfastnav/internal/globals"
)

func TestMatchAndRankByTextQuality(t *testing.T) {
	resetRankingState(t)
	resources := []globals.Resource{
		{Name: "Visual Code", Filepath: `C:\apps\code.exe`},
		{Name: "Code", Filepath: `C:\apps\exact.exe`},
		{Name: "Barcode", Filepath: `C:\apps\contains.exe`},
		{Name: "Command Editor", Filepath: `C:\apps\word.exe`},
	}

	results := MatchAndRank(resources, "code")
	if len(results) != 4 {
		t.Fatalf("got %d results, want 4", len(results))
	}
	want := []string{"Code", "Visual Code", "Barcode", "Command Editor"}
	for index, name := range want {
		if results[index].Name != name {
			t.Fatalf("result %d = %q, want %q", index, results[index].Name, name)
		}
	}
}

func TestMatchAndRankSupportsFuzzyQueries(t *testing.T) {
	resetRankingState(t)
	resources := []globals.Resource{
		{Name: "Visual Studio Code", Filepath: `C:\apps\code.exe`},
		{Name: "Notepad", Filepath: `C:\apps\notepad.exe`},
	}

	results := MatchAndRank(resources, "vsc")
	if len(results) != 1 || results[0].Name != "Visual Studio Code" {
		t.Fatalf("unexpected fuzzy results: %#v", results)
	}
}

func TestLearnedSelectionReordersSimilarMatches(t *testing.T) {
	resetRankingState(t)
	resources := []globals.Resource{
		{Name: "Alpha", Filepath: `C:\apps\alpha.exe`},
		{Name: "Alpine", Filepath: `C:\apps\alpine.exe`},
	}
	mu.Lock()
	selections = []selection{{Query: "al", Path: `c:\apps\alpine.exe`, Count: 2}}
	mu.Unlock()

	results := MatchAndRank(resources, "al")
	if len(results) != 2 || results[0].Name != "Alpine" {
		t.Fatalf("learned result was not promoted: %#v", results)
	}
}

func resetRankingState(t *testing.T) {
	t.Helper()
	mu.Lock()
	oldEntries, oldUsage, oldSelections := entries, usage, selections
	entries = nil
	usage = make(map[string]int)
	selections = nil
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		entries, usage, selections = oldEntries, oldUsage, oldSelections
		mu.Unlock()
	})
}
