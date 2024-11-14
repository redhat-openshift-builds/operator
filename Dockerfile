FROM registry.access.redhat.com/ubi9/go-toolset@sha256:b23212b6a296e95cbed5dd415d4cd5567db9e8057aa08e291f1fddc087cea944 AS builder

COPY . .

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -mod vendor -o operator cmd/main.go

FROM registry.access.redhat.com/ubi9/ubi-micro@sha256:ac53e091e8bcaf2e33877dc26fd118aea7f31ba82b7c6f083e39ad9cea6691fc

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
