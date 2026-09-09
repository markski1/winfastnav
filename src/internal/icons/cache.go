package icons

import (
	"image"
	"os"
	"strconv"
	"strings"
	"sync"
)

const maxEntries = 256

type Cache struct {
	mu         sync.Mutex
	images     map[string]image.Image
	missing    map[string]struct{}
	loading    map[string]uint64
	lastUsed   map[string]uint64
	sourceKeys map[string]string
	sequence   uint64
	limit      int
	generation uint64
	changed    func()
}

func NewCache() *Cache {
	return &Cache{
		images:     make(map[string]image.Image),
		missing:    make(map[string]struct{}),
		loading:    make(map[string]uint64),
		lastUsed:   make(map[string]uint64),
		sourceKeys: make(map[string]string),
		limit:      maxEntries,
	}
}

func (c *Cache) Image(path string) image.Image {
	baseKey, key := sourceKey(path)
	if baseKey == "" {
		return nil
	}

	c.mu.Lock()
	if previousKey := c.sourceKeys[baseKey]; previousKey != "" && previousKey != key {
		c.removeLocked(previousKey)
		c.generation++
		c.loading = make(map[string]uint64)
	}
	c.sourceKeys[baseKey] = key
	icon := c.images[key]
	_, missing := c.missing[key]
	if icon != nil || missing {
		c.touchLocked(key)
		c.mu.Unlock()
		return icon
	}
	if _, loading := c.loading[key]; !loading {
		generation := c.generation
		c.loading[key] = generation
		go c.loadAsync(path, key, generation)
	}
	c.mu.Unlock()
	return nil
}

func (c *Cache) loadAsync(path, key string, generation uint64) {
	icon := load(path)
	c.mu.Lock()
	if c.generation != generation || c.loading[key] != generation {
		c.mu.Unlock()
		return
	}
	delete(c.loading, key)
	if icon == nil {
		c.missing[key] = struct{}{}
	} else {
		c.images[key] = icon
	}
	c.touchLocked(key)
	c.evictLocked()
	changed := c.changed
	c.mu.Unlock()
	if changed != nil {
		changed()
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	c.images = make(map[string]image.Image)
	c.missing = make(map[string]struct{})
	c.loading = make(map[string]uint64)
	c.lastUsed = make(map[string]uint64)
	c.sourceKeys = make(map[string]string)
	c.sequence = 0
	c.generation++
	c.mu.Unlock()
}

func (c *Cache) Invalidate(path string) {
	baseKey := strings.ToLower(path)
	if baseKey == "" {
		return
	}
	c.mu.Lock()
	if key := c.sourceKeys[baseKey]; key != "" {
		c.removeLocked(key)
	}
	delete(c.sourceKeys, baseKey)
	c.generation++
	c.loading = make(map[string]uint64)
	c.mu.Unlock()
}

func (c *Cache) SetChangedHandler(handler func()) {
	c.mu.Lock()
	c.changed = handler
	c.mu.Unlock()
}

func (c *Cache) touchLocked(key string) {
	c.sequence++
	c.lastUsed[key] = c.sequence
}

func (c *Cache) removeLocked(key string) {
	delete(c.images, key)
	delete(c.missing, key)
	delete(c.lastUsed, key)
	if baseKey, _, ok := strings.Cut(key, "\x00"); ok && c.sourceKeys[baseKey] == key {
		delete(c.sourceKeys, baseKey)
	}
}

func (c *Cache) evictLocked() {
	for len(c.images)+len(c.missing) > c.limit {
		var oldestKey string
		var oldest uint64
		for key, used := range c.lastUsed {
			if oldestKey == "" || used < oldest {
				oldestKey = key
				oldest = used
			}
		}
		if oldestKey == "" {
			return
		}
		c.removeLocked(oldestKey)
	}
}

func sourceKey(path string) (string, string) {
	baseKey := strings.ToLower(path)
	if baseKey == "" {
		return "", ""
	}
	signature := "unknown"
	if info, err := os.Stat(path); err == nil {
		signature = strconv.FormatInt(info.ModTime().UnixNano(), 10) + ":" + strconv.FormatInt(info.Size(), 10)
	} else if os.IsNotExist(err) {
		signature = "missing"
	}
	return baseKey, baseKey + "\x00" + signature
}
