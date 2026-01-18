FROM registry.redhat.io/ubi9/go-toolset@sha256:359dd4c6c4255b3f7bce4dc15ffa5a9aa65a401f819048466fa91baa8244a793 AS builder

USER 1001

WORKDIR /opt/app-root/src

# Copy the Go Modules manifests
COPY --chown=1001:0 go.mod go.mod
COPY --chown=1001:0 go.sum go.sum

# Copy the go source
COPY --chown=1001:0 . .

ENV GOEXPERIMENT=strictfipsruntime

RUN CGO_ENABLED=1 GO111MODULE=on go build -a -mod vendor -tags strictfipsruntime -o operator cmd/main.go

FROM registry.redhat.io/ubi9-minimal@sha256:6fc28bcb6776e387d7a35a2056d9d2b985dc4e26031e98a2bd35a7137cd6fd71

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
    cpe="cpe:/a:redhat:openshift_builds:1.7::el9" \
    description="Red Hat OpenShift Builds Operator" \
    distribution-scope="public" \
    io.k8s.description="Red Hat OpenShift Builds Operator" \
    io.k8s.display-name="Red Hat OpenShift Builds Operator" \
    io.openshift.tags="builds,operator" \
    maintainer="openshift-builds@redhat.com" \
    name="openshift-builds/openshift-builds-operator-rhel9" \
    release="1" \
    summary="Red Hat OpenShift Builds Operator" \
    url="https://github.com/redhat-openshift-builds/operator" \
    vendor="Red Hat, Inc." \
    version="v1.7.0"
