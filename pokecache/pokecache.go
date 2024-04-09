package pokecache

import (
	"sync"
	"time"
)

func NewCache(interval time.Duration) Cache {
	cache := Cache{
		interval: interval,
		data:     make(map[string]cacheEntry),
		lock:     &sync.Mutex{},
	}

	go cache.reapLoop()

	return cache
}

type Cache struct {
	interval time.Duration
	data     map[string]cacheEntry
	lock     *sync.Mutex
}

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

func (c *Cache) Add(key string, val []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, ok := c.data[key]

	if !ok {
		c.data[key] = cacheEntry{
			createdAt: time.Now(),
			val:       val,
		}
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	entry, ok := c.data[key]

	if ok {
		return entry.val, ok
	}

	return make([]byte, 0), ok
}

func (c *Cache) reapLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for tick := range ticker.C {
		c.lock.Lock()
		for key, entry := range c.data {
			if entry.createdAt.Compare(tick) < 0 {
				delete(c.data, key)
			}
		}
		c.lock.Unlock()
	}
}
