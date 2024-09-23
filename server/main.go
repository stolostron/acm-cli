package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	var secure bool

	flag.BoolVar(
		&secure,
		"secure",
		false,
		"Set to true to serve with certificates over port 8443. "+
			"The default value is false to serve without certificates over port 8080.",
	)
	flag.Parse()

	port := "8080"
	directory := "/acm-cli"

	// Verify certificate and key files for TLS
	if secure {
		var missingFiles []string

		for _, crtFile := range []string{"tls.crt", "tls.key"} {
			if _, err := os.Stat("/var/run/acm-cli-cert/" + crtFile); errors.Is(err, os.ErrNotExist) {
				missingFiles = append(missingFiles, crtFile)
			}
		}

		if len(missingFiles) > 0 {
			log.Fatalf("error: Certificate files %s not found in /var/run/acm-cli-cert/\n", missingFiles)
		}

		port = "8443"
	}

	// Set up file server with timeouts
	server := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
	}

	http.Handle("/", http.FileServer(http.Dir(directory)))
	log.Printf("Serving %s on port %s\n", directory, port)

	// Serve over TLS with certificate files.
	if secure {
		log.Printf("Loaded certificate from /var/run/acm-cli-cert. Serving securely.")
		log.Fatal(
			server.ListenAndServeTLS(
				"/var/run/acm-cli-cert/tls.crt",
				"/var/run/acm-cli-cert/tls.key",
			),
		)
	}

	// Serve without certificates.
	log.Println("Serving without certificates.")
	log.Fatal(
		server.ListenAndServe(),
	)
}
