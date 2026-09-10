package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrCacheClosed = errors.New("cache is closed")
)

type Entry struct {
	Value     []byte
	ExpiresAt time.Time
}

type Stats struct {
	Hits        uint64
	Misses      uint64
	Sets        uint64
	Deletes     uint64
	Expirations uint64
	Evictions   uint64
}

type Cache struct {
	mu sync.Mutex

	lru *lru

	stopCh chan struct{}
	wg     sync.WaitGroup

	closed atomic.Bool

	hits        atomic.Uint64
	misses      atomic.Uint64
	sets        atomic.Uint64
	deletes     atomic.Uint64
	expirations atomic.Uint64
	evictions   atomic.Uint64
}

func New(capacity int) *Cache {
	if capacity <= 0 {
		capacity = 1
	}

	c := &Cache{
		lru:    newLRU(capacity),
		stopCh: make(chan struct{}),
	}

	c.wg.Add(1)
	go c.expirationWorker()

	return c
}

func (c *Cache) Set(key string, value []byte, ttl time.Duration) error {
	if c.closed.Load() {
		return ErrCacheClosed
	}

	var expiresAt time.Time

	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	// Copy the value so callers cannot mutate cached data
	// after Set returns.
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	entry := Entry{
		Value:     valueCopy,
		ExpiresAt: expiresAt,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed.Load() {
		return ErrCacheClosed
	}

	evicted := c.lru.set(key, entry)

	c.sets.Add(1)

	if evicted {
		c.evictions.Add(1)
	}

	return nil
}

func (c *Cache) Get(key string) ([]byte, error) {
	if c.closed.Load() {
		return nil, ErrCacheClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed.Load() {
		return nil, ErrCacheClosed
	}

	entry, ok := c.lru.get(key)

	if !ok {
		c.misses.Add(1)
		return nil, ErrKeyNotFound
	}

	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		c.lru.delete(key)

		c.misses.Add(1)
		c.expirations.Add(1)

		return nil, ErrKeyNotFound
	}

	value := make([]byte, len(entry.Value))
	copy(value, entry.Value)

	c.hits.Add(1)

	return value, nil
}

func (c *Cache) Delete(key string) error {
	if c.closed.Load() {
		return ErrCacheClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed.Load() {
		return ErrCacheClosed
	}

	if c.lru.delete(key) {
		c.deletes.Add(1)
	}

	return nil
}

func (c *Cache) Exists(key string) bool {
	if c.closed.Load() {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.lru.get(key)

	if !ok {
		return false
	}

	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		c.lru.delete(key)
		c.expirations.Add(1)

		return false
	}

	return true
}

func (c *Cache) Clear() error {
	if c.closed.Load() {
		return ErrCacheClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed.Load() {
		return ErrCacheClosed
	}

	count := c.lru.len()

	c.lru.clear()

	c.deletes.Add(uint64(count))

	return nil
}

func (c *Cache) Stats() Stats {
	return Stats{
		Hits:        c.hits.Load(),
		Misses:      c.misses.Load(),
		Sets:        c.sets.Load(),
		Deletes:     c.deletes.Load(),
		Expirations: c.expirations.Load(),
		Evictions:   c.evictions.Load(),
	}
}

func (c *Cache) expirationWorker() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()

		case <-c.stopCh:
			return
		}
	}
}

func (c *Cache) removeExpired() {
	if c.closed.Load() {
		return
	}

	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for key, element := range c.lru.items {
		entry := element.Value.(*lruEntry).entry

		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			c.lru.delete(key)
			c.expirations.Add(1)
		}
	}
}

func (c *Cache) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}

	close(c.stopCh)
	c.wg.Wait()
}
