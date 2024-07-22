FROM scratch

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=openshift-builds-operator
LABEL operators.operatorframework.io.bundle.channels.v1=latest,openshift-builds-1.1
LABEL operators.operatorframework.io.bundle.channel.default.v1=latest
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.35.0
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v4

# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# Labels for Red Hat Operators
LABEL com.redhat.openshift.versions="v4.12-v4.17"

# Copy files to locations specified by labels.
COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/
COPY bundle/tests/scorecard /tests/scorecard/
