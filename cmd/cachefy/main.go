package main

import (
	"log"
	"net/http"
	"time"

	"github.com/shubhams01/cachefy/internal/cache"
	cachehttp "github.com/shubhams01/cachefy/internal/http"
)

func main() {
	c := cache.New(10000)
	defer c.Close()

	handler := cachehttp.NewHandler(c)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("Cachefy listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}
}
