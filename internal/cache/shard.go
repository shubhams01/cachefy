package cache

import (
	"sync"
	"time"
)

type shard struct {
	mu sync.Mutex

	lru *lru
}

func newShard(capacity int) *shard {
	return &shard{
		lru: newLRU(capacity),
	}
}

func (s *shard) set(key string, entry Entry) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.lru.set(key, entry)
}

func (s *shard) get(key string) (Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.lru.get(key)
}

func (s *shard) delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.lru.delete(key)
}

func (s *shard) exists(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.lru.get(key)

	return ok
}

func (s *shard) clear() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := s.lru.len()

	s.lru.clear()

	return count
}

func (s *shard) removeExpired(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0

	for key, element := range s.lru.items {
		entry := element.Value.(*lruEntry).entry

		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			s.lru.delete(key)
			count++
		}
	}

	return count
}
