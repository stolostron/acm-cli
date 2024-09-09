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
BUILD_DIR ?= ./build/_output

.PHONY: build
build:
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/acm-cli-server ./server/main.go

.PHONY: build-image
build-image:
	$(CONTAINER_ENGINE) build --platform linux/$(ARCH) $(BUILD_ARGS) -t $(IMAGE_NAME_AND_VERSION):$(TAG) .

.PHONY: package-binaries
package-binaries:
	@cp LICENSE $(BUILD_DIR)
	# Packaging Windows binaries into zip archives
	cd $(BUILD_DIR) && find . -type f \
		-name "*.exe" \
		-exec bash -c 'mkdir $$(basename {} ".exe")' \; \
		-exec bash -c 'mv {} $$(basename {} ".exe")' \; \
		-exec bash -c 'cp LICENSE $$(basename {} ".exe")' \; \
		-exec bash -c 'zip -rv $$(basename {} ".exe").zip $$(basename {} ".exe")' \; \
		-exec bash -c 'rm -r $$(basename {} ".exe")' \;
	# Packaging Linux/Darwin binaries into tarballs
	cd $(BUILD_DIR) && find . -type f \
		-not -name "acm-cli-server" \
		-not -name "*.zip" \
		-not -name "*.tar.gz" \
		-not -name "LICENSE" \
		-exec tar -zvcf {}.tar.gz {} LICENSE \; \
		-exec rm {} \;
	@rm $(BUILD_DIR)/LICENSE

############################################################
# deploy section
############################################################

.PHONY: deploy-openshift
deploy-openshift:
	# Preflight check for OCP cluster
	@if ! (kubectl get ingresses.config.openshift.io 1>/dev/null); then \
		echo "info: Unable to fetch Ingress, or not an Openshift cluster. Exiting."; exit 1; \
	fi
	# Creating the acm-cli-downloads deployment with ConsoleCLIDownload in namespace default
	helm template deploy/ \
		--set isOpenshift=true --set imagePullPolicy=IfNotPresent \
		--set ingress.domain=$$(kubectl get ingresses.config.openshift.io cluster -o jsonpath={.spec.domain}) \
		| oc apply -n default -f -

.PHONY: deploy
deploy:
	# Creating the acm-cli-downloads deployment in namespace default
	helm template deploy/ | kubectl apply -n default -f -
	kubectl rollout status deployment -n default acm-cli-downloads

############################################################
# lint section
############################################################

.PHONY: fmt
fmt:

.PHONY: lint
lint:

############################################################
# test section
############################################################
CLUSTER_NAME = acm-cli

.PHONY: kind-bootstrap-cluster
kind-bootstrap-cluster: KIND_ARGS = --config test/kind_config.yaml
kind-bootstrap-cluster: kind-create-cluster
	kind load --name $(KIND_NAME) docker-image quay.io/stolostron/acm-cli:latest
	$(MAKE) deploy

.PHONY: e2e-test
e2e-test: kind-bootstrap-cluster
	# Checking availability of CLI server endpoint
	for i in {1..5}; do \
		curl -sS http://localhost:30000 1>/dev/null || \
			{ echo "Connection failed. Retrying ($${i}/5)"; sleep 3; }; \
	done
	# Validating returned file list against test/cli_list.html
	curl -s http://localhost:30000 | diff test/cli_list.html - && echo "Success!"
