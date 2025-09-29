FROM scratch

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=openshift-builds-operator
LABEL operators.operatorframework.io.bundle.channels.v1=latest
LABEL operators.operatorframework.io.bundle.channel.default.v1=latest
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.39.2
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v4

LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

LABEL com.redhat.openshift.versions="v4.16-v4.19" \
    com.redhat.component="openshift-builds-operator-bundle-container" \
    description="Red Hat OpenShift Builds Operator Bundle" \
    distribution-scope="public" \
    io.k8s.description="Red Hat OpenShift Builds Operator Bundle" \
    io.k8s.display-name="Red Hat OpenShift Builds Operator Bundle" \
    io.openshift.tags="builds,operator,bundle" \
    maintainer="openshift-builds@redhat.com" \
    name="openshift-builds/openshift-builds-operator-bundle" \
    summary="Red Hat OpenShift Builds Operator Bundle" \
    url="https://catalog.redhat.com/software/containers/openshift-builds/openshift-builds-operator-bundle" \
    vendor="Red Hat, Inc." \
    version="1.5.1" \
    release="1.5.1" \
    cpe="cpe:/a:redhat:openshift_builds:1.5::el9"

COPY bundle/ /
COPY LICENSE /licenses/

USER 65532:65532
