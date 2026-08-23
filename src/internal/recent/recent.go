package recent

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"winfastnav/internal/globals"
	"winfastnav/internal/settings"
)

const maxEntries = 12

var (
	mu      sync.RWMutex
	entries []string
)

func Load() {
	stored, err := settings.GetSetting("recent")
	if err != nil || stored == "" {
		return
	}

	var loaded []string
	if json.Unmarshal([]byte(stored), &loaded) != nil {
		return
	}
	if len(loaded) > maxEntries {
		loaded = loaded[:maxEntries]
	}

	mu.Lock()
	entries = loaded
	mu.Unlock()
}

func Record(path string) {
	if path == "" {
		return
	}

	mu.Lock()
	updated := make([]string, 0, maxEntries)
	updated = append(updated, path)
	for _, entry := range entries {
		if !strings.EqualFold(entry, path) && len(updated) < maxEntries {
			updated = append(updated, entry)
		}
	}
	entries = updated
	stored, err := json.Marshal(entries)
	mu.Unlock()
	if err == nil {
		_ = settings.SetSetting("recent", string(stored))
	}
}

func Rank(resources []globals.Resource) []globals.Resource {
	mu.RLock()
	order := make(map[string]int, len(entries))
	for index, entry := range entries {
		order[strings.ToLower(entry)] = index
	}
	mu.RUnlock()

	sort.SliceStable(resources, func(i, j int) bool {
		left, leftRecent := order[strings.ToLower(resources[i].Filepath)]
		right, rightRecent := order[strings.ToLower(resources[j].Filepath)]
		switch {
		case leftRecent && rightRecent:
			return left < right
		case leftRecent:
			return true
		case rightRecent:
			return false
		default:
			return false
		}
	})
	return resources
}

func Only(resources []globals.Resource) []globals.Resource {
	mu.RLock()
	order := make(map[string]int, len(entries))
	for index, entry := range entries {
		order[strings.ToLower(entry)] = index
	}
	mu.RUnlock()

	var recent []globals.Resource
	for _, resource := range resources {
		if _, ok := order[strings.ToLower(resource.Filepath)]; ok {
			recent = append(recent, resource)
		}
	}
	return Rank(recent)
}
