FROM registry.redhat.io/ubi9/ubi-minimal@sha256:14f14e03d68f7fd5f2b18a13478b6b127c341b346c86b6e0b886ed2b7573b8e0

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=openshift-builds-operator
LABEL operators.operatorframework.io.bundle.channels.v1=latest,openshift-builds-1.1
LABEL operators.operatorframework.io.bundle.channel.default.v1=latest
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.35.0
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v4

LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

LABEL com.redhat.openshift.versions="v4.14-v4.18" \
    com.redhat.component="openshift-builds-operator-bundle-container" \
    description="Red Hat OpenShift Builds Operator Bundle" \
    distribution-scope="public" \
    io.k8s.description="Red Hat OpenShift Builds Operator Bundle" \
    io.k8s.display-name="Red Hat OpenShift Builds Operator Bundle" \
    io.openshift.tags="builds,operator,bundle" \
    maintainer="openshift-builds@redhat.com" \
    name="openshift-builds/operator-bundle" \
    summary="Red Hat OpenShift Builds Operator Bundle" \
    url="https://catalog.redhat.com/software/containers/openshift-builds/openshift-builds-operator-bundle" \
    vendor="Red Hat, Inc." \
    version="v1.3.0"

COPY bundle/ /
COPY LICENSE /licenses/

USER 65532:65532
