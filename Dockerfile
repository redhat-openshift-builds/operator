FROM registry.access.redhat.com/ubi9/go-toolset@sha256:97e30a01caeece72ee967013e7c7af777ea4ee93840681ddcfe38a87eb4c084a AS builder

COPY . .

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -mod vendor -o operator cmd/main.go

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
