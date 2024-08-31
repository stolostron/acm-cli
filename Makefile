PWD := $(shell pwd)
LOCAL_BIN ?= $(PWD)/bin

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.x

# Keep an existing GOPATH, make a private one if it is undefined
GOPATH_DEFAULT := $(PWD)/.go
export GOPATH ?= $(GOPATH_DEFAULT)
GOBIN_DEFAULT := $(GOPATH)/bin
export GOBIN ?= $(GOBIN_DEFAULT)
export PATH := $(LOCAL_BIN):$(GOBIN):$(PATH)

# Default architecture to amd64
ARCH ?= $(shell go env GOARCH)
ifndef ARCH
	ARCH = amd64
endif

include build/common/Makefile.common.mk

# Image URL to use all building/pushing image targets;
# Use your own docker registry and image name for dev/test by overridding the IMG and REGISTRY environment variable.
IMG ?= acm-cli
REGISTRY ?= quay.io/stolostron
TAG ?= latest
IMAGE_NAME_AND_VERSION ?= $(REGISTRY)/$(IMG)

############################################################
# clean section
############################################################

.PHONY: clean
clean:
	-rm build/_output/*
	-rm kubeconfig_*
	-rm gosec.json
	-rm kubeconfig_*
	kind delete cluster --name $(KIND_NAME)

############################################################
# build section
############################################################
CONTAINER_ENGINE ?= podman

.PHONY: build
build:
	CGO_ENABLED=1 go build -o ./build/_output/acm-cli-server ./server/main.go

.PHONY: build-image
build-image:
	$(CONTAINER_ENGINE) build --platform linux/$(ARCH) $(BUILD_ARGS) -t $(IMAGE_NAME_AND_VERSION):$(TAG) .

