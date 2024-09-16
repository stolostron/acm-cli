FROM registry.ci.openshift.org/stolostron/builder:go1.22-linux AS builder

ENV RELEASE_TAG=release-2.12 \
    REPO_PATH=/go/src/github.com/stolostron/acm-cli

WORKDIR ${REPO_PATH}

# Build the HTTP server binary
COPY . .
RUN make build

# Fetch and package imported binaries
RUN make build-and-package

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ENV REPO_PATH=/go/src/github.com/stolostron/acm-cli

RUN microdnf update -y \
    && microdnf clean all

# Copy binaries from builder
COPY --from=builder ${REPO_PATH}/build/_output/* /acm-cli/
RUN mv /acm-cli/acm-cli-server /usr/local/bin/

# Run as non-root user
USER 1001

ENTRYPOINT [ "/usr/local/bin/acm-cli-server" ]
