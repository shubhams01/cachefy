package cache

import (
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c := New(10)
	defer c.Close()

	err := c.Set("name", []byte("cachefy"), 0)

	if err != nil {
		t.Fatal(err)
	}

	value, err := c.Get("name")

	if err != nil {
		t.Fatal(err)
	}

	if string(value) != "cachefy" {
		t.Fatalf("expected cachefy, got %s", string(value))
	}
}

func TestCacheGetMissing(t *testing.T) {
	c := New(10)
	defer c.Close()

	_, err := c.Get("missing")

	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestCacheDelete(t *testing.T) {
	c := New(10)
	defer c.Close()

	c.Set("key", []byte("value"), 0)

	err := c.Delete("key")

	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Get("key")

	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestCacheExists(t *testing.T) {
	c := New(10)
	defer c.Close()

	c.Set("key", []byte("value"), 0)

	if !c.Exists("key") {
		t.Fatal("key should exist")
	}

	c.Delete("key")

	if c.Exists("key") {
		t.Fatal("key should not exist")
	}
}

func TestCacheTTL(t *testing.T) {
	c := New(10)
	defer c.Close()

	c.Set("key", []byte("value"), 50*time.Millisecond)

	_, err := c.Get("key")

	if err != nil {
		t.Fatal("key should exist before expiration")
	}

	time.Sleep(100 * time.Millisecond)

	_, err = c.Get("key")

	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("expected key to expire, got %v", err)
	}
}

func TestCacheValueIsolation(t *testing.T) {
	c := New(10)
	defer c.Close()

	value := []byte("original")

	c.Set("key", value, 0)

	value[0] = 'X'

	cached, err := c.Get("key")

	if err != nil {
		t.Fatal(err)
	}

	if string(cached) != "original" {
		t.Fatalf("expected original, got %s", string(cached))
	}

	cached[0] = 'Y'

	cachedAgain, err := c.Get("key")

	if err != nil {
		t.Fatal(err)
	}

	if string(cachedAgain) != "original" {
		t.Fatalf("expected original, got %s", string(cachedAgain))
	}
}

func TestCacheClear(t *testing.T) {
	c := New(10)
	defer c.Close()

	c.Set("a", []byte("A"), 0)
	c.Set("b", []byte("B"), 0)
	c.Set("c", []byte("C"), 0)

	err := c.Clear()

	if err != nil {
		t.Fatal(err)
	}

	if c.Exists("a") || c.Exists("b") || c.Exists("c") {
		t.Fatal("cache should be empty")
	}
}

func TestLRUEviction(t *testing.T) {
	c := New(2)
	defer c.Close()

	c.Set("a", []byte("A"), 0)
	c.Set("b", []byte("B"), 0)

	_, err := c.Get("a")

	if err != nil {
		t.Fatal("a should exist")
	}

	c.Set("c", []byte("C"), 0)

	_, err = c.Get("b")

	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatal("b should have been evicted")
	}

	_, err = c.Get("a")

	if err != nil {
		t.Fatal("a should still exist")
	}

	_, err = c.Get("c")

	if err != nil {
		t.Fatal("c should exist")
	}
}

func TestLRUUpdate(t *testing.T) {
	c := New(2)
	defer c.Close()

	c.Set("a", []byte("A"), 0)
	c.Set("b", []byte("B"), 0)

	c.Set("a", []byte("updated"), 0)

	c.Set("c", []byte("C"), 0)

	_, err := c.Get("b")

	if !errors.Is(err, ErrKeyNotFound) {
		t.Fatal("b should have been evicted")
	}

	value, err := c.Get("a")

	if err != nil {
		t.Fatal(err)
	}

	if string(value) != "updated" {
		t.Fatalf("expected updated, got %s", string(value))
	}
}

func TestCacheStats(t *testing.T) {
	c := New(10)
	defer c.Close()

	c.Set("key", []byte("value"), 0)

	c.Get("key")
	c.Get("missing")
	c.Delete("key")

	stats := c.Stats()

	if stats.Sets != 1 {
		t.Fatalf("expected 1 set, got %d", stats.Sets)
	}

	if stats.Hits != 1 {
		t.Fatalf("expected 1 hit, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Fatalf("expected 1 miss, got %d", stats.Misses)
	}

	if stats.Deletes != 1 {
		t.Fatalf("expected 1 delete, got %d", stats.Deletes)
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	c := New(1000)
	defer c.Close()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			key := "key"

			for j := 0; j < 100; j++ {
				_ = c.Set(key, []byte("value"), 0)
				_, _ = c.Get(key)
				_ = c.Exists(key)
			}
		}(i)
	}

	wg.Wait()
}

func TestCacheClose(t *testing.T) {
	c := New(10)

	c.Close()
	c.Close()

	err := c.Set("key", []byte("value"), 0)

	if !errors.Is(err, ErrCacheClosed) {
		t.Fatalf("expected ErrCacheClosed, got %v", err)
	}

	_, err = c.Get("key")

	if !errors.Is(err, ErrCacheClosed) {
		t.Fatalf("expected ErrCacheClosed, got %v", err)
	}
}

func TestCacheShardDistribution(t *testing.T) {
	c := New(320)
	defer c.Close()

	if len(c.shards) != 32 {
		t.Fatalf("expected 32 shards, got %d", len(c.shards))
	}

	for i := 0; i < 320; i++ {
		key := "key-" + strconv.Itoa(i)

		if err := c.Set(key, []byte("value"), 0); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 320; i++ {
		key := "key-" + strconv.Itoa(i)

		if !c.Exists(key) {
			t.Fatalf("expected %s to exist", key)
		}
	}
}
