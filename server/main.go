package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	directory := "/acm-cli"

	// Set up file server with timeouts
	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
	}

	http.Handle("/", http.FileServer(http.Dir(directory)))
	log.Printf("Serving %s on port 8080\n", directory)

	log.Fatal(server.ListenAndServe())
}
