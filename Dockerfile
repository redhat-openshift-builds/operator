# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset:1.21.11-7@sha256:d128c3d36878251f039606f144ef2608746c3203708b722295e6c571df1d8613 AS builder
ARG TARGETOS
ARG TARGETARCH

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# This operator vendors dependencies. Copy this layer first to take advantage of cache hits.
COPY vendor/ vendor/

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o operator cmd/main.go

# Use Red Hat Universal Base Image Micro (aka UBI Distroless) to package the manager binary
# Refer to https://catalog.redhat.com/software/containers/ubi9/ubi-micro/615bdf943f6014fa45ae1b58
FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:7f376b75faf8ea546f28f8529c37d24adcde33dca4103f4897ae19a43d58192b

WORKDIR /

COPY --from=builder /opt/app-root/src/operator .
COPY config/shipwright/ config/shipwright/
COPY config/sharedresource/ config/sharedresource/
COPY LICENSE /licenses/

USER 65532:65532

ENTRYPOINT ["/operator"]

LABEL \
    com.redhat.component="openshift-builds-operator-container" \
    name="openshift-builds/operator" \
    version="v1.1.0" \
    summary="Red Hat OpenShift Builds Operator" \
    maintainer="openshift-builds@redhat.com" \
    description="Red Hat OpenShift Builds Operator" \
    io.k8s.description="Red Hat OpenShift Builds Operator" \
    io.k8s.display-name="Red Hat OpenShift Builds Operator" \
    io.openshift.tags="builds,operator"
