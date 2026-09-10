package recent

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"winfastnav/internal/globals"
	"winfastnav/internal/settings"
)

const (
	maxEntries    = 12
	maxSelections = 200
)

type selection struct {
	Query string `json:"query"`
	Path  string `json:"path"`
	Count int    `json:"count"`
	Last  int64  `json:"last"`
}

var (
	mu         sync.RWMutex
	entries    []string
	usage      map[string]int
	selections []selection
)

func Load() {
	loadedUsage := make(map[string]int)
	var loadedSelections []selection

	stored, err := settings.GetSetting("recent")
	var loaded []string
	if err == nil && stored != "" {
		_ = json.Unmarshal([]byte(stored), &loaded)
	}
	if len(loaded) > maxEntries {
		loaded = loaded[:maxEntries]
	}

	if stored, err = settings.GetSetting("usage"); err == nil && stored != "" {
		_ = json.Unmarshal([]byte(stored), &loadedUsage)
	}

	if stored, err = settings.GetSetting("query_selections"); err == nil && stored != "" {
		_ = json.Unmarshal([]byte(stored), &loadedSelections)
		if len(loadedSelections) > maxSelections {
			loadedSelections = loadedSelections[:maxSelections]
		}
	}

	mu.Lock()
	entries = loaded
	usage = loadedUsage
	selections = loadedSelections
	mu.Unlock()
}

func Record(path string) {
	if path == "" {
		return
	}

	mu.Lock()
	if usage == nil {
		usage = make(map[string]int)
	}
	usage[strings.ToLower(path)]++
	updated := make([]string, 0, maxEntries)
	updated = append(updated, path)
	for _, entry := range entries {
		if !strings.EqualFold(entry, path) && len(updated) < maxEntries {
			updated = append(updated, entry)
		}
	}
	entries = updated
	storedRecent, recentErr := json.Marshal(entries)
	storedUsage, usageErr := json.Marshal(usage)
	mu.Unlock()
	if recentErr == nil {
		_ = settings.SetSetting("recent", string(storedRecent))
	}
	if usageErr == nil {
		_ = settings.SetSetting("usage", string(storedUsage))
	}
}

func RecordSelection(query, path string) {
	query = normalizeQuery(query)
	path = strings.ToLower(strings.TrimSpace(path))
	if query == "" || path == "" {
		return
	}

	mu.Lock()
	now := time.Now().UnixNano()
	updated := make([]selection, 0, min(len(selections)+1, maxSelections))
	match := selection{Query: query, Path: path, Count: 1, Last: now}
	for _, item := range selections {
		if item.Query == query && strings.EqualFold(item.Path, path) {
			match.Count = item.Count + 1
			continue
		}
		updated = append(updated, item)
	}
	updated = append(updated, match)
	sort.SliceStable(updated, func(i, j int) bool { return updated[i].Last > updated[j].Last })
	if len(updated) > maxSelections {
		updated = updated[:maxSelections]
	}
	selections = updated
	stored, err := json.Marshal(selections)
	mu.Unlock()
	if err == nil {
		_ = settings.SetSetting("query_selections", string(stored))
	}
}

func MatchAndRankLimit(resources []globals.Resource, query string, limit int) []globals.Resource {
	return matchAndRank(resources, query, limit)
}

func matchAndRank(resources []globals.Resource, query string, limit int) []globals.Resource {
	query = normalizeQuery(query)
	if query == "" {
		result := rank(resources)
		if limit > 0 && len(result) > limit {
			result = result[:limit]
		}
		return result
	}

	mu.RLock()
	recency := make(map[string]int, len(entries))
	for index, entry := range entries {
		recency[strings.ToLower(entry)] = index
	}
	selectionCounts := make(map[string]int)
	for _, item := range selections {
		if item.Query == query {
			selectionCounts[strings.ToLower(item.Path)] = item.Count
		}
	}

	type ranked struct {
		resource globals.Resource
		score    int
	}
	rankedResources := make([]ranked, 0, len(resources))
	for _, resource := range resources {
		score, matched := matchScore(resource, query)
		if !matched {
			continue
		}
		path := resource.SearchPath
		if path == "" {
			path = strings.ToLower(resource.Filepath)
		}
		if count := selectionCounts[path]; count > 0 {
			score += 6000 + min(count, 20)*100
		}
		if count := usage[path]; count > 0 {
			score += min(count, 50) * 20
		}
		if index, ok := recency[path]; ok {
			score += 500 - index*25
		}
		rankedResources = append(rankedResources, ranked{resource: resource, score: score})
	}
	mu.RUnlock()

	sort.SliceStable(rankedResources, func(i, j int) bool {
		if rankedResources[i].score != rankedResources[j].score {
			return rankedResources[i].score > rankedResources[j].score
		}
		left := rankedResources[i].resource.SearchName
		if left == "" {
			left = strings.ToLower(rankedResources[i].resource.Name)
		}
		right := rankedResources[j].resource.SearchName
		if right == "" {
			right = strings.ToLower(rankedResources[j].resource.Name)
		}
		return left < right
	})
	resultCount := len(rankedResources)
	if limit > 0 && resultCount > limit {
		resultCount = limit
	}
	result := make([]globals.Resource, resultCount)
	for index, item := range rankedResources[:resultCount] {
		result[index] = item.resource
	}
	return result
}

func matchScore(resource globals.Resource, query string) (int, bool) {
	name := resource.SearchName
	if name == "" {
		name = strings.ToLower(strings.TrimSpace(resource.Name))
	}
	path := resource.SearchPath
	if path == "" {
		path = strings.ToLower(resource.Filepath)
	}
	switch {
	case name == query:
		return 100000, true
	case strings.HasPrefix(name, query):
		return 80000 - (len(name) - len(query)), true
	case hasWordPrefix(name, query):
		return 65000, true
	case strings.Contains(name, query):
		return 50000 - strings.Index(name, query), true
	case fuzzyMatch(name, query):
		return 30000, true
	case strings.Contains(path, query):
		return 10000 - strings.Index(path, query), true
	default:
		return 0, false
	}
}

func hasWordPrefix(name, query string) bool {
	wordStart := true
	for index := 0; index < len(name); index++ {
		if wordStart && strings.HasPrefix(name[index:], query) {
			return true
		}
		wordStart = name[index] == ' ' || name[index] == '-' || name[index] == '_' || name[index] == '.'
	}
	return false
}

func fuzzyMatch(name, query string) bool {
	queryOffset := 0
	wanted, size := utf8.DecodeRuneInString(query)
	for _, candidate := range name {
		if candidate == wanted {
			queryOffset += size
			if queryOffset == len(query) {
				return true
			}
			wanted, size = utf8.DecodeRuneInString(query[queryOffset:])
		}
	}
	return false
}

func normalizeQuery(query string) string {
	return strings.ToLower(strings.Join(strings.Fields(query), " "))
}

func rank(resources []globals.Resource) []globals.Resource {
	mu.RLock()
	order := make(map[string]int, len(entries))
	for index, entry := range entries {
		order[strings.ToLower(entry)] = index
	}
	mu.RUnlock()
	if len(order) == 0 {
		return resources
	}

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
	return rank(recent)
}
