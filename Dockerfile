FROM registry.access.redhat.com/ubi9/go-toolset@sha256:703937e152d049e62f5aa8ab274a4253468ab70f7b790d92714b37cf0a140555 AS builder

COPY . .

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -mod vendor -o operator cmd/main.go

FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:f6e0a71b7e0875b54ea559c2e0a6478703268a8d4b8bdcf5d911d0dae76aef51

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
