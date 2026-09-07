package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	g "winfastnav/internal/globals"
)

type Settings map[string]string

var (
	settingsMu sync.Mutex
	settings   Settings
)

func SetupSettings() {
	var blocklist []string
	stored, _ := GetSetting("blocklist")
	if stored != "" {
		if err := json.Unmarshal([]byte(stored), &blocklist); err != nil {
			log.Printf("Error parsing blocklist: %v", err)
			if err = SetSetting("blocklist", "[]"); err != nil {
				log.Printf("Error resetting blocklist: %v", err)
			}
		}
	}
	g.ExecBlocklist = blocklist

	g.SearchString, _ = GetSetting("searchstring")
	if g.SearchString == "" {
		g.SearchString = "https://duckduckgo.com/?q=%s"
		if err := SetSetting("searchstring", g.SearchString); err != nil {
			log.Printf("Error setting searchstring: %v", err)
		}
	}
	g.AliasString, _ = GetSetting("aliases")
}

func getSettingsFilePath() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", errors.New("can't find appdata")
	}

	dir := filepath.Join(appData, "winfastnav")

	err := os.MkdirAll(dir, 0o700)
	if err != nil {
		return "", fmt.Errorf("failed to create app directory: %w", err)
	}

	return filepath.Join(dir, "prefs.json"), nil
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

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err = enc.Encode(s); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return nil
}

func SetSetting(key, value string) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	if err := loadSettings(); err != nil {
		return err
	}
	settings[key] = value
	return writeSettings(settings)
}

func GetSetting(key string) (string, error) {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	if err := loadSettings(); err != nil {
		return "", err
	}
	return settings[key], nil
}

func loadSettings() error {
	if settings != nil {
		return nil
	}
	loaded, err := readSettings()
	if err == nil {
		settings = loaded
	}
	return err
}
