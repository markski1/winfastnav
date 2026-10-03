package settings

import "testing"

func TestAsyncSettingsFlushesLatestValues(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	SetSettingsAsync(Settings{"recent": "first", "usage": "count"})
	SetSettingsAsync(Settings{"recent": "latest"})
	if err := Flush(); err != nil {
		t.Fatal(err)
	}

	saved, err := readSettings()
	if err != nil {
		t.Fatal(err)
	}
	if saved["recent"] != "latest" || saved["usage"] != "count" {
		t.Fatalf("saved settings = %#v", saved)
	}
}

func TestFailedSaveDoesNotChangeSettings(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("APPDATA", directory)
	if err := SetSetting("theme", "dark"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPDATA", "")
	if err := SetSettings(Settings{"theme": "light", "new": "unsaved"}); err == nil {
		t.Fatal("expected a save error")
	}
	if GetSetting("theme") != "dark" || GetSetting("new") != "" {
		t.Fatal("failed save changed cached settings")
	}
	t.Setenv("APPDATA", directory)
	if err := SetSetting("density", "compact"); err != nil {
		t.Fatal(err)
	}
	saved, err := readSettings()
	if err != nil {
		t.Fatal(err)
	}
	if saved["theme"] != "dark" || saved["new"] != "" || saved["density"] != "compact" {
		t.Fatalf("later save persisted failed changes: %#v", saved)
	}
}
