package main

import (
	"fmt"
	"time"

	"github.com/shubhams01/cachefy/internal/cache"
)

func main() {
	c := cache.New()

	c.Set(
		"message",
		[]byte("Hello from Cachefy"),
		10*time.Second,
	)

	value, err := c.Get("message")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(value))
}