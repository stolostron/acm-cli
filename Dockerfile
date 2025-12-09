FROM registry.ci.openshift.org/stolostron/builder:go1.25-linux AS builder

ENV REPO_PATH=/go/src/github.com/stolostron/acm-cli

WORKDIR ${REPO_PATH}

# Build the HTTP server binary
COPY . .
RUN make build

# Fetch and package imported binaries
RUN make sync-build-package

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ENV REPO_PATH=/go/src/github.com/stolostron/acm-cli

RUN microdnf install -y tar

# Copy binaries from builder
COPY --from=builder ${REPO_PATH}/build/_output/* /acm-cli/
RUN mv /acm-cli/acm-cli-server /usr/local/bin/

# Copy license
RUN mkdir licenses/
COPY LICENSE licenses/

# Run as non-root user
USER 1001

ENTRYPOINT [ "/usr/local/bin/acm-cli-server" ]

LABEL name="rhacm2/acm-cli-rhel9"
LABEL summary="Serve ACM CLI binaries"
LABEL description="Serve ACM CLI binaries through the Red Hat Openshift console"
LABEL io.k8s.display-name="ACM CLI downloads"
LABEL io.k8s.description="Serve ACM CLI binaries through the Red Hat Openshift console"
LABEL com.redhat.component="acm-cli-container"
LABEL io.openshift.tags="data,images"
