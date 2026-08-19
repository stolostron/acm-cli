// Copyright (c) 2026 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	openshifttls "github.com/openshift/controller-runtime-common/pkg/tls"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	sdktls "open-cluster-management.io/sdk-go/pkg/tls"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// scheme only registers the OpenShift resource this server needs (the cluster APIServer config),
// rather than the whole config.openshift.io API group.
var scheme = k8sruntime.NewScheme() //nolint:gochecknoglobals

func init() {
	scheme.AddKnownTypes(configv1.GroupVersion, &configv1.APIServer{}, &configv1.APIServerList{})
	metav1.AddToGroupVersion(scheme, configv1.GroupVersion)

	// This server logs through the standard "log" package rather than logr; silence
	// controller-runtime's own logging instead of standing up a second logging stack.
	ctrl.SetLogger(logr.Discard())
}

// resolveEffectiveTLSConfig returns the function to apply to the HTTPS server's tls.Config,
// whether it's worth starting the drift watcher, the resolved TLS profile spec (used to seed
// that watcher), and whether valid --tls-min-version/--tls-cipher-suites flags were actually
// applied (as opposed to being absent or rejected as invalid), in order of precedence:
//  1. Explicit --tls-min-version/--tls-cipher-suites flags.
//  2. The OpenShift APIServer "cluster" resource's tlsSecurityProfile, when present.
//  3. Go's TLS defaults.
func resolveEffectiveTLSConfig(
	ctx context.Context, apiServerClient client.Client, minVersion, cipherSuites string,
) (tlsOptsFunc func(*tls.Config), watchAPIServer bool, resolvedSpec configv1.TLSProfileSpec, flagsApplied bool) {
	flagCfg, err := sdktls.ConfigFromFlags(minVersion, cipherSuites)
	if err != nil {
		log.Printf("error: invalid --tls-min-version/--tls-cipher-suites, ignoring the flags: %v", err)

		flagCfg = nil
	}

	if flagCfg != nil {
		log.Printf("Effective TLS configuration determined from flags: minVersion=%s cipherSuites=%s",
			sdktls.VersionToString(flagCfg.MinVersion), sdktls.CipherSuitesToString(flagCfg.CipherSuites))

		return sdktls.ConfigToFunc(flagCfg), false, configv1.TLSProfileSpec{}, true
	}

	spec, found, watchable := fetchAPIServerTLSProfileSpec(ctx, apiServerClient)
	if !found {
		log.Println("Effective TLS configuration using Go defaults")

		return func(*tls.Config) {}, watchable, configv1.TLSProfileSpec{}, false
	}

	tlsConfigFunc, unsupported := openshifttls.NewTLSConfigFromProfile(spec)
	if len(unsupported) > 0 {
		log.Printf("warning: some cipher suites/groups in the APIServer TLS profile are not "+
			"implemented by Go's crypto/tls package and will be ignored: %v", unsupported)
	}

	log.Printf("Effective TLS configuration determined from the OpenShift APIServer cluster "+
		"resource: minVersion=%s ciphers=%v groups=%v", spec.MinTLSVersion, spec.Ciphers, spec.Groups)

	return tlsConfigFunc, watchable, spec, false
}

// fetchAPIServerTLSProfileSpec returns the resolved profile spec when found. watchable is false
// only when the CRD/resource definitively doesn't exist; it stays true on other errors (e.g.
// RBAC, connectivity) since those might be transient and the drift watcher can recover once they
// clear.
func fetchAPIServerTLSProfileSpec(
	ctx context.Context, apiServerClient client.Client,
) (spec configv1.TLSProfileSpec, found, watchable bool) {
	if apiServerClient == nil {
		return configv1.TLSProfileSpec{}, false, false
	}

	spec, err := openshifttls.FetchAPIServerTLSProfile(ctx, apiServerClient)
	switch {
	case err == nil:
		return spec, true, true
	case apierrors.IsNotFound(err):
		log.Println("OpenShift APIServer cluster resource not found")

		return configv1.TLSProfileSpec{}, false, false
	case meta.IsNoMatchError(err):
		log.Println("OpenShift APIServer CRD not installed")

		return configv1.TLSProfileSpec{}, false, false
	default:
		log.Printf("error: failed to read the OpenShift APIServer cluster resource; will keep "+
			"watching it in case this was transient: %v", err)

		return configv1.TLSProfileSpec{}, false, true
	}
}

// startAPIServerTLSWatcher runs a minimal controller-runtime manager in the background whose only
// job is to exit the process when the APIServer TLS security profile changes, so that a Deployment
// restart picks up the new settings. This MUST only be invoked when the cluster might have the
// resource (i.e. not definitively CRD-less) and no overriding flags are set.
func startAPIServerTLSWatcher(ctx context.Context, cfg *rest.Config, initialSpec configv1.TLSProfileSpec) error {
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	if err != nil {
		return fmt.Errorf("failed to create the TLS security profile watcher manager: %w", err)
	}

	watcher := &openshifttls.SecurityProfileWatcher{
		Client:                mgr.GetClient(),
		InitialTLSProfileSpec: initialSpec,
		OnProfileChange: func(_ context.Context, _, _ configv1.TLSProfileSpec) {
			log.Println("The OpenShift APIServer TLS security profile changed; exiting to apply the new settings")

			os.Exit(0)
		},
	}

	if err := watcher.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed to set up the TLS security profile watcher: %w", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			log.Printf("error: TLS security profile watcher stopped unexpectedly: %v", err)
		}
	}()

	return nil
}
