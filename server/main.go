// Copyright (c) 2026 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	openshifttls "github.com/openshift/controller-runtime-common/pkg/tls"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// shutdownTimeout bounds how long the server waits for in-flight requests (e.g. large binary
// downloads) to finish draining before forcing a shutdown.
const shutdownTimeout = 15 * time.Second

func main() {
	var secure bool

	var tlsMinVersion, tlsCipherSuites string

	flag.BoolVar(
		&secure,
		"secure",
		false,
		"Set to true to serve with certificates over port 8443. "+
			"The default value is false to serve without certificates over port 8080.",
	)
	flag.StringVar(
		&tlsMinVersion,
		"tls-min-version",
		"",
		"The minimum TLS version for the HTTPS server (e.g. VersionTLS12). "+
			"Overrides the OpenShift APIServer TLS security profile, if any is detected.",
	)
	flag.StringVar(
		&tlsCipherSuites,
		"tls-cipher-suites",
		"",
		"A comma-separated list of IANA TLS cipher suite names for the HTTPS server. "+
			"Overrides the OpenShift APIServer TLS security profile, if any is detected.",
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
		ctx := ctrl.SetupSignalHandler()

		var apiServerClient client.Client

		cfg, err := ctrl.GetConfig()
		if err != nil {
			log.Printf("warning: unable to load a Kubernetes client config, skipping OpenShift "+
				"APIServer TLS profile detection: %v", err)
		} else if apiServerClient, err = client.New(cfg, client.Options{Scheme: scheme}); err != nil {
			log.Printf("warning: failed to create a client for the OpenShift APIServer resource: %v", err)
		}

		tlsOptsFunc, watchAPIServer, initialSpec, tlsOverrideActive := resolveEffectiveTLSConfig(
			ctx, apiServerClient, tlsMinVersion, tlsCipherSuites,
		)

		// MinVersion is an explicit floor: tlsOptsFunc overrides it with the flag- or
		// APIServer-profile-derived version whenever one was resolved above.
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		tlsOptsFunc(tlsConfig)
		openshifttls.SetNextProtos(openshifttls.HTTP1NextProtos...)(tlsConfig)
		server.TLSConfig = tlsConfig

		if watchAPIServer && !tlsOverrideActive {
			if err := startAPIServerTLSWatcher(ctx, cfg, initialSpec); err != nil {
				log.Printf("error: unable to start the TLS security profile watcher; the server "+
					"will not pick up TLS profile changes without a manual restart: %v", err)
			}
		}

		serverErrCh := make(chan error, 1)

		go func() {
			serverErrCh <- server.ListenAndServeTLS(
				"/var/run/acm-cli-cert/tls.crt",
				"/var/run/acm-cli-cert/tls.key",
			)
		}()

		log.Printf("Loaded certificate from /var/run/acm-cli-cert. Serving securely.")

		select {
		case err := <-serverErrCh:
			log.Fatalf("error: server exited unexpectedly: %v", err)
		case <-ctx.Done():
			log.Println("Received shutdown signal, closing the server")

			// ctx is already cancelled here; a fresh, bounded context is needed to give
			// in-flight requests a window to drain instead of an immediate hard close.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("error: graceful shutdown did not complete cleanly: %v", err)
			}

			cancel()

			if err := <-serverErrCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("error: unexpected error after shutdown: %v", err)
			}
		}

		return
	}

	// Serve without certificates.
	log.Println("Serving without certificates.")
	log.Fatal(
		server.ListenAndServe(),
	)
}
