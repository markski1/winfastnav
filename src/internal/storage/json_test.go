package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONPreservesExistingFileOnFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "prefs.json")
	for _, value := range []string{"original", "updated"} {
		if err := WriteJSON(path, map[string]string{"value": value}, "  "); err != nil {
			t.Fatal(err)
		}
	}
	if err := WriteJSON(path, make(chan int), ""); err == nil {
		t.Fatal("expected an encoding error")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]string
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["value"] != "updated" {
		t.Fatalf("saved data = %s", data)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "prefs.json" {
		t.Fatalf("temporary files left behind: %v", entries)
	}
}
