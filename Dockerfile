FROM registry.redhat.io/ubi10/go-toolset@sha256:557f358c83892a8b94fb72649a99e1c303a625d240547fb97a9cbc1620e808c0 AS builder

USER 1001

WORKDIR /opt/app-root/src

# Copy the Go Modules manifests
COPY --chown=1001:0 go.mod go.mod
COPY --chown=1001:0 go.sum go.sum

# Copy the go source
COPY --chown=1001:0 . .

ENV GOEXPERIMENT=strictfipsruntime

RUN CGO_ENABLED=1 GO111MODULE=on go build -a -mod vendor -tags strictfipsruntime -o operator cmd/main.go

FROM registry.redhat.io/ubi10-minimal@sha256:552f8cc1ddb0bc9e411065c78499aa4dbc2c4ea9bd057aa0e6e44940db13acb5

WORKDIR /

COPY --from=builder /opt/app-root/src /opt/app-root/src
COPY --from=builder /opt/app-root/src/operator .
COPY config/shipwright/ config/shipwright/
COPY config/sharedresource/ config/sharedresource/
COPY config/networkpolicies/ config/networkpolicies/
COPY LICENSE /licenses/

USER 65532:65532

ENTRYPOINT ["/operator"]

LABEL \
    com.redhat.component="openshift-builds-operator" \
    cpe="cpe:/a:redhat:openshift_builds:1.9::el10" \
    description="Red Hat OpenShift Builds Operator" \
    distribution-scope="public" \
    io.k8s.description="Red Hat OpenShift Builds Operator" \
    io.k8s.display-name="Red Hat OpenShift Builds Operator" \
    io.openshift.tags="builds,operator" \
    maintainer="openshift-builds@redhat.com" \
    name="openshift-builds/openshift-builds-rhel10-operator" \
    release="1" \
    summary="Red Hat OpenShift Builds Operator" \
    url="https://github.com/redhat-openshift-builds/operator" \
    vendor="Red Hat, Inc." \
    version="v1.9.0"
