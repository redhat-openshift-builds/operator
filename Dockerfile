FROM registry.redhat.io/ubi10/go-toolset@sha256:cb712e78c45e429142e846c1b8b91894970eab3b66fd5a95d336b65b4f7bb877 AS builder

USER 1001

WORKDIR /opt/app-root/src

# Copy the Go Modules manifests
COPY --chown=1001:0 go.mod go.mod
COPY --chown=1001:0 go.sum go.sum

# Copy the go source
COPY --chown=1001:0 . .

ENV GOEXPERIMENT=strictfipsruntime

RUN CGO_ENABLED=1 GO111MODULE=on go build -a -mod vendor -tags strictfipsruntime -o operator cmd/main.go

FROM registry.redhat.io/ubi10-minimal@sha256:c039ecb9ac9db2db85e78b332d66521f6ce401cf812006a27bc7827018eac4a2

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
