package cacheinmemory_v2

import (
	"sync"
	"time"
)

type cacheItem struct {
	value     any
	expiredAt time.Time
}

var ttl time.Duration

type CacheInMemoryV2 struct {
	data     map[string]cacheItem
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func New(ttlValue time.Duration) *CacheInMemoryV2 {
	ttl = ttlValue
	cache := &CacheInMemoryV2{
		data:     make(map[string]cacheItem),
		stopChan: make(chan struct{}),
	}

	cache.wg.Add(1)
	go cache.cleanupExpired()

	return cache
}

func (c *CacheInMemoryV2) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cacheItem{
		value,
		time.Now().Add(ttl),
	}
}

func (c *CacheInMemoryV2) Get(key string) any {
	c.mu.RLock()
	dataFromCache := c.data[key]
	c.mu.RUnlock()

	if dataFromCache.value == nil {
		return nil
	}
	if time.Now().After(dataFromCache.expiredAt) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return nil
	}
	return dataFromCache.value
}

func (c *CacheInMemoryV2) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *CacheInMemoryV2) cleanupExpired() {
	defer c.wg.Done()

	cleanupInterval := ttl / 2
	if cleanupInterval < 100*time.Millisecond {
		cleanupInterval = 100 * time.Millisecond
	}

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, item := range c.data {
				if now.After(item.expiredAt) {
					delete(c.data, key)
				}
			}
			c.mu.Unlock()
		case <-c.stopChan:
			return
		}
	}
}

func (c *CacheInMemoryV2) Stop() {
	close(c.stopChan)
	c.wg.Wait()
}
