package main

import (
	"fmt"
	"time"

	"github.com/shubhams01/cachefy/internal/cache"
)

func main() {
	c := cache.New(100)

	defer c.Close()

	err := c.Set(
		"hello",
		[]byte("Cachefy"),
		5*time.Minute,
	)

	if err != nil {
		panic(err)
	}

	value, err := c.Get("hello")

	if err != nil {
		panic(err)
	}

	fmt.Println(string(value))
}
