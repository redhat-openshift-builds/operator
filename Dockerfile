FROM registry.access.redhat.com/ubi9/go-toolset@sha256:8a9e1465ae487fa2b4a94d2baa249a8d59fb3d7e6884a7b22c783f494f4cfb01 AS builder

COPY . .

ENV GOEXPERIMENT=strictfipsruntime

RUN CGO_ENABLED=1 GO111MODULE=on go build -a -mod vendor -tags strictfipsruntime -o operator cmd/main.go

FROM registry.redhat.io/ubi9/ubi-minimal@sha256:14f14e03d68f7fd5f2b18a13478b6b127c341b346c86b6e0b886ed2b7573b8e0

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
    version="v1.3.0" \
    summary="Red Hat OpenShift Builds Operator" \
    maintainer="openshift-builds@redhat.com" \
    description="Red Hat OpenShift Builds Operator" \
    io.k8s.description="Red Hat OpenShift Builds Operator" \
    io.k8s.display-name="Red Hat OpenShift Builds Operator" \
    io.openshift.tags="builds,operator"
