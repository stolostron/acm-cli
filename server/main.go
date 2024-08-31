package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	secure := true
	port := "8443"
	directory := "/acm-cli"

	// Verify certificate and key files for TLS
	for _, crtFile := range []string{"tls.crt", "tls.key"} {
		if _, err := os.Stat("/var/run/acm-cli-cert/" + crtFile); errors.Is(err, os.ErrNotExist) {
			log.Printf("warn: Certificate file %s not found in /var/run/acm-cli-cert/", crtFile)

			secure = false
			port = "8080"
		}
	}

	// Set up file server with timeouts
	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
	}

	http.Handle("/", http.FileServer(http.Dir(directory)))
	log.Printf("Serving %s on HTTP port: %s\n", directory, port)

	// Serve over TLS if certificate files are available.
	if secure {
		log.Printf("Loaded certificate from /var/run/acm-cli-cert. Serving securely.")
		log.Fatal(
			server.ListenAndServeTLS(
				"/var/run/acm-cli-cert/tls.crt",
				"/var/run/acm-cli-cert/tls.key",
			),
		)
	}

	// Serve over HTTP--expected certificate files were not found.
	log.Println("Certificates failed to load. Serving over HTTP.")
	log.Fatal(
		server.ListenAndServe(),
	)
}
