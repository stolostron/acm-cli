FROM registry.ci.openshift.org/stolostron/builder:go1.22-linux AS builder

ENV RELEASE_TAG=release-2.12 \
    REPO_PATH=/go/src/github.com/stolostron/acm-cli

WORKDIR ${REPO_PATH}

# Build the HTTP server binary
COPY . .
RUN make build

# Fetch and build the Policy toolset
RUN git clone --branch=${RELEASE_TAG} --depth=1 \
        https://github.com/stolostron/policy-cli
RUN cd policy-cli && \
        make build-release && \
        mv build/_output/policy-cli ${REPO_PATH}/build/_output/

# Fetch and build the Policy generator
RUN git clone --branch=${RELEASE_TAG} --depth=1 \
        https://github.com/stolostron/policy-generator-plugin
RUN cd policy-generator-plugin && \
        make build-binary &&  make build-release && \
        mv PolicyGenerator ${REPO_PATH}/build/_output/ && \
        mv  build_output/* ${REPO_PATH}/build/_output/

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ENV REPO_PATH=/go/src/github.com/stolostron/acm-cli \
    USER_UID=1001 \
    USER_NAME=acm-cli

RUN microdnf update -y \
    && microdnf clean all

# Copy binaries from builder
COPY --from=builder ${REPO_PATH}/build/_output/* /acm-cli/
RUN mv /acm-cli/acm-cli-server /usr/local/bin/

# Setup non-root user
COPY --from=builder ${REPO_PATH}/build/user_setup /usr/local/bin/
RUN  /usr/local/bin/user_setup
USER 1001

ENTRYPOINT [ "/usr/local/bin/acm-cli-server" ]
