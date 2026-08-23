package icons

import (
	"image"
	"strings"
	"sync"
)

type Cache struct {
	mu      sync.RWMutex
	images  map[string]image.Image
	missing map[string]struct{}
}

func NewCache() *Cache {
	return &Cache{
		images:  make(map[string]image.Image),
		missing: make(map[string]struct{}),
	}
}

func (c *Cache) Image(path string) image.Image {
	key := strings.ToLower(path)
	if key == "" {
		return nil
	}

	c.mu.RLock()
	image := c.images[key]
	_, missing := c.missing[key]
	c.mu.RUnlock()
	if image != nil || missing {
		return image
	}

	image = load(path)
	c.mu.Lock()
	if image == nil {
		c.missing[key] = struct{}{}
	} else {
		c.images[key] = image
	}
	c.mu.Unlock()
	return image
}
