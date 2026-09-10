package cache

import (
	"errors"
	"sync"
	"time"
)

var ErrKeyNotFound = errors.New("cache: key not found")

type Entry struct {
	Value     []byte
	ExpiresAt time.Time
}

type Cache struct {
	mu   sync.RWMutex
	data map[string]Entry
}

func New() *Cache {
	return &Cache{
		data: make(map[string]Entry),
	}
}

func (c *Cache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time

	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	c.data[key] = Entry{
		Value:     append([]byte(nil), value...),
		ExpiresAt: expiresAt,
	}
}

func (c *Cache) Get(key string) ([]byte, error) {
	c.mu.RLock()

	entry, ok := c.data[key]

	if !ok {
		c.mu.RUnlock()
		return nil, ErrKeyNotFound
	}

	if !entry.ExpiresAt.IsZero() &&
		time.Now().After(entry.ExpiresAt) {

		c.mu.RUnlock()

		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()

		return nil, ErrKeyNotFound
	}

	value := append([]byte(nil), entry.Value...)

	c.mu.RUnlock()

	return value, nil
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
}

func (c *Cache) Exists(key string) bool {
	_, err := c.Get(key)
	return err == nil
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]Entry)
}