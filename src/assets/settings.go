package assets

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Settings map[string]string

func SetupSettings() {
	unparsedList, err := GetSetting("blocklist")
	if err != nil || len(unparsedList) == 0 {
		// initialize with empty list string
		err = SetSetting("blocklist", "[]")
		if err != nil {
			log.Printf("Error setting blocklist: %v", err)
			return
		}
		unparsedList = "[]"
	}

	var blocklist []string
	err = json.Unmarshal([]byte(unparsedList), &blocklist)
	if err != nil {
		log.Printf("Error parsing blocklist: %v", err)
		return
	}

	ExecBlocklist = blocklist

	SearchString, err = GetSetting("searchstring")
	if err != nil || len(SearchString) == 0 {
		// initialize with empty list string
		err = SetSetting("searchstring", "https://duckduckgo.com/?q=")
		if err != nil {
			log.Printf("Error setting searchstring: %v", err)
			return
		}
		SearchString = "https://duckduckgo.com/?q="
	}
}

func getSettingsFilePath() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", errors.New("can't find appdata")
	}

	dir := filepath.Join(appData, "winfastnav")

	err := os.MkdirAll(dir, os.ModePerm)
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

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing settings file: %v", err)
		}
	}(file)

	var s Settings
	dec := json.NewDecoder(file)
	err = dec.Decode(&s)
	if err != nil {
		return nil, err
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

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing settings file: %v", err)
		}
	}(file)

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	err = enc.Encode(s)
	if err != nil {
		return err
	}

	return nil
}

// SetSetting stores a key-value pair in the settings and persists it to file.
func SetSetting(key, value string) error {
	settings, err := readSettings()
	if err != nil {
		return err
	}

	settings[key] = value

	return writeSettings(settings)
}

// GetSetting retrieves the value for a given key from settings.
// Returns (value, true) if found, or ("", false) if the key does not exist.
func GetSetting(key string) (string, error) {
	settings, err := readSettings()

	if err != nil {
		return "", err
	}

	value := settings[key]
	return value, nil
}
