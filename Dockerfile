FROM registry.redhat.io/ubi9/go-toolset:1.24 AS builder

USER 1001

WORKDIR /opt/app-root/src

# Copy the Go Modules manifests
COPY --chown=1001:0 go.mod go.mod
COPY --chown=1001:0 go.sum go.sum

# Copy the go source
COPY --chown=1001:0 . .

ENV GOEXPERIMENT=strictfipsruntime

RUN CGO_ENABLED=1 GO111MODULE=on go build -a -mod vendor -tags strictfipsruntime -o operator cmd/main.go

FROM registry.redhat.io/ubi9/ubi-minimal:9.6

WORKDIR /

COPY --from=builder /opt/app-root/src /opt/app-root/src
COPY --from=builder /opt/app-root/src/operator .
COPY config/shipwright/ config/shipwright/
COPY config/sharedresource/ config/sharedresource/
COPY LICENSE /licenses/

USER 65532:65532

ENTRYPOINT ["/operator"]

LABEL \
    com.redhat.component="openshift-builds-operator-container" \
    name="openshift-builds/openshift-builds-operator" \
    version="v1.5.2" \
    summary="Red Hat OpenShift Builds Operator" \
    maintainer="openshift-builds@redhat.com" \
    description="Red Hat OpenShift Builds Operator" \
    io.k8s.description="Red Hat OpenShift Builds Operator" \
    io.k8s.display-name="Red Hat OpenShift Builds Operator" \
    io.openshift.tags="builds,operator" \
    distribution-scope="public" \
    url="https://catalog.redhat.com/en/software/containers/openshift-builds/openshift-builds-rhel9-operator" \
    vendor="Red Hat, Inc." \
    release="1.5.2" \
    cpe="cpe:/a:redhat:openshift_builds:1.5::el9"
