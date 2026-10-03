package icons

import (
	"image"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxEntries = 256
const checkInterval = time.Minute

type Cache struct {
	mu         sync.Mutex
	images     map[string]image.Image
	missing    map[string]struct{}
	loading    map[string]uint64
	lastUsed   map[string]uint64
	signatures map[string]string
	nextCheck  map[string]time.Time
	checking   map[string]bool
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
		signatures: make(map[string]string),
		nextCheck:  make(map[string]time.Time),
		checking:   make(map[string]bool),
		limit:      maxEntries,
	}
}

func (c *Cache) Image(path string) image.Image {
	key := strings.ToLower(path)
	if key == "" {
		return nil
	}

	c.mu.Lock()
	icon := c.images[key]
	_, missing := c.missing[key]
	if icon != nil || missing {
		c.touchLocked(key)
		if !c.checking[key] && time.Now().After(c.nextCheck[key]) {
			c.checking[key] = true
			go c.refreshAsync(path, key, c.generation)
		}
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
	signature := sourceSignature(path)
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
	c.signatures[key] = signature
	c.nextCheck[key] = time.Now().Add(checkInterval)
	c.touchLocked(key)
	c.evictLocked()
	changed := c.changed
	c.mu.Unlock()
	if changed != nil {
		changed()
	}
}

func (c *Cache) refreshAsync(path, key string, generation uint64) {
	signature := sourceSignature(path)
	c.mu.Lock()
	if c.generation != generation || !c.checking[key] {
		c.mu.Unlock()
		return
	}
	delete(c.checking, key)
	if signature == c.signatures[key] {
		c.nextCheck[key] = time.Now().Add(checkInterval)
		c.mu.Unlock()
		return
	}
	c.removeLocked(key)
	c.loading[key] = generation
	c.mu.Unlock()
	go c.loadAsync(path, key, generation)
}

func (c *Cache) Clear() {
	c.mu.Lock()
	c.images = make(map[string]image.Image)
	c.missing = make(map[string]struct{})
	c.loading = make(map[string]uint64)
	c.lastUsed = make(map[string]uint64)
	c.signatures = make(map[string]string)
	c.nextCheck = make(map[string]time.Time)
	c.checking = make(map[string]bool)
	c.sequence = 0
	c.generation++
	c.mu.Unlock()
}

func (c *Cache) Invalidate(path string) {
	key := strings.ToLower(path)
	if key == "" {
		return
	}
	c.mu.Lock()
	c.removeLocked(key)
	c.generation++
	c.loading = make(map[string]uint64)
	c.checking = make(map[string]bool)
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
	delete(c.signatures, key)
	delete(c.nextCheck, key)
	delete(c.checking, key)
}

func sourceSignature(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "missing"
		}
		return "unknown"
	}
	return strconv.FormatInt(info.ModTime().UnixNano(), 10) + ":" + strconv.FormatInt(info.Size(), 10)
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
