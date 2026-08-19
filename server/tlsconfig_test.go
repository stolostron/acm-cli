// Copyright (c) 2026 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package main

import (
	"context"
	"crypto/tls"
	"reflect"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	openshifttls "github.com/openshift/controller-runtime-common/pkg/tls"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

// applyToConfig runs a TLSOpts function against a fresh tls.Config and returns it, to make
// assertions on the result easier to read.
func applyToConfig(f func(*tls.Config)) *tls.Config {
	cfg := &tls.Config{} //nolint:gosec // test-only, never used to actually serve traffic

	f(cfg)

	return cfg
}

func TestFetchAPIServerTLSProfileSpecNotFound(t *testing.T) {
	c := fakeclient.NewClientBuilder().WithScheme(scheme).Build()

	spec, found, watchable := fetchAPIServerTLSProfileSpec(t.Context(), c)

	if found {
		t.Fatal("expected found to be false")
	}

	if watchable {
		t.Fatal("expected watchable to be false: no CRD/resource means there's nothing to watch")
	}

	if !reflect.DeepEqual(spec, configv1.TLSProfileSpec{}) {
		t.Fatalf("expected an empty spec, got %+v", spec)
	}
}

func TestFetchAPIServerTLSProfileSpecFound(t *testing.T) {
	apiServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: openshifttls.APIServerName},
		Spec: configv1.APIServerSpec{
			TLSSecurityProfile: &configv1.TLSSecurityProfile{Type: configv1.TLSProfileModernType},
		},
	}

	c := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(apiServer).Build()

	spec, found, watchable := fetchAPIServerTLSProfileSpec(t.Context(), c)

	if !found {
		t.Fatal("expected found to be true")
	}

	if !watchable {
		t.Fatal("expected watchable to be true")
	}

	want := *configv1.TLSProfiles[configv1.TLSProfileModernType]
	if !reflect.DeepEqual(spec, want) {
		t.Fatalf("expected spec %+v, got %+v", want, spec)
	}
}

// TestFetchAPIServerTLSProfileSpecForbiddenStaysWatchable proves that an RBAC-denied Get is
// treated as possibly transient: found is false (fall back to defaults now), but watchable stays
// true so the drift watcher still gets registered. If the permission is later granted, the
// watcher's informer will sync, see the real profile differs from the zero-value spec it was
// seeded with, and trigger a restart to pick it up -- without any dedicated retry loop.
func TestFetchAPIServerTLSProfileSpecForbiddenStaysWatchable(t *testing.T) {
	c := fakeclient.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(
				_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption,
			) error {
				return apierrors.NewForbidden(configv1.Resource("apiservers"), openshifttls.APIServerName, nil)
			},
		}).
		Build()

	spec, found, watchable := fetchAPIServerTLSProfileSpec(t.Context(), c)

	if found {
		t.Fatal("expected found to be false")
	}

	if !watchable {
		t.Fatal("expected watchable to be true: an RBAC/connectivity error might be transient")
	}

	if !reflect.DeepEqual(spec, configv1.TLSProfileSpec{}) {
		t.Fatalf("expected an empty spec, got %+v", spec)
	}
}

func TestFetchAPIServerTLSProfileSpecNilClient(t *testing.T) {
	spec, found, watchable := fetchAPIServerTLSProfileSpec(t.Context(), nil)

	if found {
		t.Fatal("expected found to be false")
	}

	if watchable {
		t.Fatal("expected watchable to be false: no client means there's nothing to watch")
	}

	if !reflect.DeepEqual(spec, configv1.TLSProfileSpec{}) {
		t.Fatalf("expected an empty spec, got %+v", spec)
	}
}

func TestResolveEffectiveTLSConfigFlagsTakePrecedence(t *testing.T) {
	// Flags must be honored without ever consulting the (nil, in this test) API server client.
	tlsOptsFunc, watchAPIServer, spec, flagsApplied := resolveEffectiveTLSConfig(t.Context(), nil, "VersionTLS13", "")

	if watchAPIServer {
		t.Fatal("expected watchAPIServer to be false")
	}

	if !flagsApplied {
		t.Fatal("expected flagsApplied to be true: valid flags were provided")
	}

	if !reflect.DeepEqual(spec, configv1.TLSProfileSpec{}) {
		t.Fatalf("expected an empty spec, got %+v", spec)
	}

	if got := applyToConfig(tlsOptsFunc).MinVersion; got != tls.VersionTLS13 {
		t.Fatalf("expected MinVersion %d, got %d", tls.VersionTLS13, got)
	}
}

func TestResolveEffectiveTLSConfigInvalidFlagFallsBackToGoDefaults(t *testing.T) {
	tlsOptsFunc, watchAPIServer, spec, flagsApplied := resolveEffectiveTLSConfig(
		t.Context(), nil, "not-a-real-version", "",
	)

	if watchAPIServer {
		t.Fatal("expected watchAPIServer to be false")
	}

	if flagsApplied {
		t.Fatal("expected flagsApplied to be false: the flag was invalid and should have been ignored")
	}

	if !reflect.DeepEqual(spec, configv1.TLSProfileSpec{}) {
		t.Fatalf("expected an empty spec, got %+v", spec)
	}

	if got := applyToConfig(tlsOptsFunc); !reflect.DeepEqual(got, &tls.Config{}) { //nolint:gosec // test-only
		t.Fatalf("expected an unmodified tls.Config, got %+v", got)
	}
}

// TestResolveEffectiveTLSConfigInvalidFlagKeepsWatcherWhenAPIServerWatchable guards against a
// regression where an invalid, non-empty flag was mistaken for an active override even after
// being ignored, which suppressed the drift watcher and left the server unable to pick up later
// APIServer TLS profile changes. flagsApplied must reflect whether a flag override was actually
// used, not merely whether one was attempted.
func TestResolveEffectiveTLSConfigInvalidFlagKeepsWatcherWhenAPIServerWatchable(t *testing.T) {
	apiServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: openshifttls.APIServerName},
		Spec: configv1.APIServerSpec{
			TLSSecurityProfile: &configv1.TLSSecurityProfile{Type: configv1.TLSProfileModernType},
		},
	}

	c := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(apiServer).Build()

	tlsOptsFunc, watchAPIServer, spec, flagsApplied := resolveEffectiveTLSConfig(
		t.Context(), c, "not-a-real-version", "",
	)

	if flagsApplied {
		t.Fatal("expected flagsApplied to be false: the flag was invalid and should have been ignored")
	}

	if !watchAPIServer {
		t.Fatal("expected watchAPIServer to be true: an invalid flag must not suppress watching a " +
			"watchable APIServer resource")
	}

	want := *configv1.TLSProfiles[configv1.TLSProfileModernType]
	if !reflect.DeepEqual(spec, want) {
		t.Fatalf("expected spec %+v, got %+v", want, spec)
	}

	if got := applyToConfig(tlsOptsFunc).MinVersion; got != tls.VersionTLS13 {
		t.Fatalf("expected the modern profile's MinVersion %d, got %d", tls.VersionTLS13, got)
	}
}

func TestResolveEffectiveTLSConfigNoOverrideNoAPIServer(t *testing.T) {
	// With no flags and no client config, this should fall back to Go defaults without a watcher.
	tlsOptsFunc, watchAPIServer, spec, flagsApplied := resolveEffectiveTLSConfig(t.Context(), nil, "", "")

	if watchAPIServer {
		t.Fatal("expected watchAPIServer to be false: no client config means there's nothing to watch")
	}

	if flagsApplied {
		t.Fatal("expected flagsApplied to be false: no flags were provided")
	}

	if !reflect.DeepEqual(spec, configv1.TLSProfileSpec{}) {
		t.Fatalf("expected an empty spec, got %+v", spec)
	}

	if got := applyToConfig(tlsOptsFunc); !reflect.DeepEqual(got, &tls.Config{}) { //nolint:gosec // test-only
		t.Fatalf("expected an unmodified tls.Config, got %+v", got)
	}
}
