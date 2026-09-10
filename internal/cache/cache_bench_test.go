package cache

import (
	"strconv"
	"testing"
	"time"
)

func BenchmarkCacheSet(b *testing.B) {
	c := New(b.N)
	defer c.Close()

	value := []byte("cachefy")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Set("key", value, time.Minute)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	c := New(1)
	defer c.Close()

	_ = c.Set("key", []byte("cachefy"), time.Minute)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = c.Get("key")
	}
}

func BenchmarkCacheSetParallel(b *testing.B) {
	c := New(1000)
	defer c.Close()

	value := []byte("cachefy")

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = c.Set("key", value, time.Minute)
		}
	})
}

func BenchmarkCacheGetParallel(b *testing.B) {
	c := New(1000)
	defer c.Close()

	_ = c.Set("key", []byte("cachefy"), time.Minute)

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = c.Get("key")
		}
	})
}

func BenchmarkCacheShardedSetParallel(b *testing.B) {
	c := New(100000)
	defer c.Close()

	value := []byte("cachefy")

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0

		for pb.Next() {
			key := "key-" + strconv.Itoa(i)
			_ = c.Set(key, value, time.Minute)
			i++
		}
	})
}

func BenchmarkCacheShardedGetParallel(b *testing.B) {
	c := New(100000)
	defer c.Close()

	for i := 0; i < 1000; i++ {
		key := "key-" + strconv.Itoa(i)
		_ = c.Set(key, []byte("cachefy"), time.Minute)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0

		for pb.Next() {
			key := "key-" + strconv.Itoa(i%1000)
			_, _ = c.Get(key)
			i++
		}
	})
}
