FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_1.23 AS builder

COPY . .

ENV GOEXPERIMENT=strictfipsruntime

RUN CGO_ENABLED=1 GO111MODULE=on go build -a -mod vendor -tags strictfipsruntime -o operator cmd/main.go

FROM registry.redhat.io/ubi9/ubi-minimal@sha256:34880b64c07f28f64d95737f82f891516de9a3b43583f39970f7bf8e4cfa48b7

WORKDIR /

COPY --from=builder /operator .
COPY config/shipwright/ config/shipwright/
COPY config/sharedresource/ config/sharedresource/
COPY LICENSE /licenses/

USER 65532:65532

ENTRYPOINT ["/operator"]

LABEL \
    com.redhat.component="openshift-builds-operator-container" \
    name="openshift-builds/operator" \
    version="v1.4.1" \
    summary="Red Hat OpenShift Builds Operator" \
    maintainer="openshift-builds@redhat.com" \
    description="Red Hat OpenShift Builds Operator" \
    io.k8s.description="Red Hat OpenShift Builds Operator" \
    io.k8s.display-name="Red Hat OpenShift Builds Operator" \
    io.openshift.tags="builds,operator"
