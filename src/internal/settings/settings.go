package settings

import (
	"encoding/json"
	"errors"
	"log"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"

	g "winfastnav/internal/globals"
	"winfastnav/internal/storage"
)

type Settings map[string]string

var (
	settingsMu sync.Mutex
	settings   Settings
	dirty      bool
	writeOnce  sync.Once
	writes     = make(chan struct{}, 1)
)

func SetupSettings() {
	var blocklist []string
	stored := GetSetting("blocklist")
	if stored != "" {
		if err := json.Unmarshal([]byte(stored), &blocklist); err != nil {
			log.Printf("Error parsing blocklist: %v", err)
			if err = SetSetting("blocklist", "[]"); err != nil {
				log.Printf("Error resetting blocklist: %v", err)
			}
		}
	}
	g.ExecBlocklist = blocklist

	g.SearchString = GetSetting("searchstring")
	if g.SearchString == "" {
		g.SearchString = "https://duckduckgo.com/?q=%s"
		if err := SetSetting("searchstring", g.SearchString); err != nil {
			log.Printf("Error setting searchstring: %v", err)
		}
	}
}

func getSettingsFilePath() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", errors.New("can't find appdata")
	}

	return filepath.Join(appData, "winfastnav", "prefs.json"), nil
}

func readSettings() (Settings, error) {
	path, err := getSettingsFilePath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)

	if err != nil {
		if os.IsNotExist(err) {
			return Settings{}, nil // No settings yet
		}
		return nil, err
	}

	defer file.Close()

	var s Settings
	dec := json.NewDecoder(file)
	err = dec.Decode(&s)
	if err != nil {
		return nil, err
	}
	if s == nil {
		s = Settings{}
	}

	return s, nil
}

func writeSettings(s Settings) error {
	path, err := getSettingsFilePath()
	if err != nil {
		return err
	}

	return storage.WriteJSON(path, s, "  ")
}

func SetSetting(key, value string) error {
	return SetSettings(Settings{key: value})
}

func SetSettings(values Settings) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	loadSettings()
	next := maps.Clone(settings)
	maps.Copy(next, values)
	if err := writeSettings(next); err != nil {
		return err
	}
	settings, dirty = next, false
	return nil
}

func SetSettingsAsync(values Settings) {
	settingsMu.Lock()
	loadSettings()
	for key, value := range values {
		settings[key] = value
	}
	dirty = true
	settingsMu.Unlock()

	writeOnce.Do(func() { go writePendingSettings() })
	select {
	case writes <- struct{}{}:
	default:
	}
}

func Flush() error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	return flushLocked()
}

func flushLocked() error {
	if !dirty {
		return nil
	}
	if err := writeSettings(settings); err != nil {
		return err
	}
	dirty = false
	return nil
}

func writePendingSettings() {
	for range writes {
		timer := time.NewTimer(300 * time.Millisecond)
		for pending := true; pending; {
			select {
			case <-writes:
				timer.Reset(300 * time.Millisecond)
			case <-timer.C:
				if err := Flush(); err != nil {
					log.Printf("failed to save settings: %v", err)
				}
				pending = false
			}
		}
	}
}

func GetSetting(key string) string {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	loadSettings()
	return settings[key]
}

func loadSettings() {
	if settings != nil {
		return
	}
	loaded, err := readSettings()
	if err != nil {
		log.Printf("failed to read settings, starting with defaults: %v", err)
		settings = Settings{}
		return
	}
	settings = loaded
}
