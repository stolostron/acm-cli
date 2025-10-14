# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

The `acm-cli` repository provides a collection of binaries for managing Red Hat Advanced Cluster Management for Kubernetes (RHACM), delivered through a container image containing a Golang web server. The binaries are packaged and served to make them readily downloadable through Kubernetes/Openshift consoles.

## Architecture

### Core Components

1. **HTTP File Server** (`server/main.go`)
   - Simple Go web server that serves static files from `/acm-cli` directory
   - Supports both HTTP (port 8080) and HTTPS (port 8443) modes
   - TLS certificates expected at `/var/run/acm-cli-cert/` when using `--secure` flag
   - Main entry point for the container

2. **Git Submodules** (`external/`)
   - `policy-cli`: Toolset for managing RHACM policies (from stolostron/policy-cli)
   - `policy-generator-plugin`: Kustomize plugin to generate RHACM policies (from stolostron/policy-generator-plugin)
   - Tracked in `.gitmodules` with branch references (currently `release-2.15`)

3. **Build System**
   - `build/cli_map.csv`: Defines which external repos to build and package
   - `build/cli-builder.sh`: Builds binaries from submodules based on cli_map.csv
   - `build/cli-packager.sh`: Packages binaries into platform-specific archives (.tar.gz for Linux/Darwin, .zip for Windows)
   - Outputs to `build/_output/`

4. **Deployment** (`deploy/`)
   - Helm charts for Kubernetes deployment
   - Supports both generic Kubernetes (NodePort over HTTP) and OpenShift (with Route and ConsoleCLIDownload)
   - Templates include deployment, service, route, and consoleclidownload manifests

### Build Flow

```
1. Sync submodules (git submodule update --init)
2. Build binaries from external repos (cli-builder.sh reads cli_map.csv)
3. Package binaries for multiple platforms (cli-packager.sh)
4. Build container image with HTTP server + packaged binaries
5. Deploy to cluster via Helm
```

The container image uses a multi-stage build:
- Builder stage: Builds the HTTP server and fetches/packages imported binaries
- Runtime stage: UBI9-minimal with packaged binaries and server binary

## Common Development Commands

### Building

```bash
# Build the HTTP server binary only
make build

# Build container image locally
make build-image

# Sync git submodules and build/package all binaries
make sync-build-package

# Or run steps individually:
make sync-repos          # Initialize/update submodules
make build-binaries      # Build from external repos
make package-binaries    # Package into archives
```

### Deployment

```bash
# Deploy to generic Kubernetes cluster (NodePort/HTTP)
make deploy

# Deploy to OpenShift with Route and ConsoleCLIDownload
make deploy-openshift
```

### Testing

```bash
# End-to-end test: Deploy to Kind cluster and verify served files
make e2e-test
# This validates the served file list against test/cli_list.html
```

### Cleanup

```bash
make clean  # Removes build outputs, test artifacts, and Kind cluster
```

## Important Patterns

### Submodule Branch Matching

The `cli-builder.sh` script enforces branch consistency:
- If the parent repo is on a `release-*` branch, submodules must be on matching release branches
- Branch is configured in `.gitmodules` for each submodule
- This ensures version alignment across the repositories

### Container Image Configuration

- Default image: `quay.io/stolostron/acm-cli:latest`
- Override via environment variables: `REGISTRY`, `IMG`, `TAG`
- Default architecture: `amd64`, override with `ARCH` env var
- Container engine: `podman` by default, override with `CONTAINER_ENGINE`

### Served File Structure

Binaries are packaged and served as:
- `{os}-{arch}-{tool}.tar.gz` (Linux/Darwin)
- `{os}-{arch}-{tool}.zip` (Windows)

Supported platforms:
- darwin-amd64, darwin-arm64
- linux-amd64, linux-arm64
- windows-amd64, windows-arm64

## Testing Notes

The e2e test deploys to a Kind cluster with:
- NodePort service exposed on port 30000
- File server accessibility validated via curl
- Served file list compared against `test/cli_list.html`

## Go Version

Currently using Go 1.24 (see `go.mod` and Dockerfile builder image reference).
