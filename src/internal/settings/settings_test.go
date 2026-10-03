package settings

import "testing"

func TestAsyncSettingsFlushesLatestValues(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	if err := SetSettingsAsync(Settings{"recent": "first", "usage": "count"}); err != nil {
		t.Fatal(err)
	}
	if err := SetSettingsAsync(Settings{"recent": "latest"}); err != nil {
		t.Fatal(err)
	}
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
