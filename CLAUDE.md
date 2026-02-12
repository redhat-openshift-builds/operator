# OpenShift Builds Operator - Project Standards

## Project Overview

This operator deploys and manages **Shipwright Build** components and the **Shared Resource CSI Driver** on OpenShift clusters. It is built with Go 1.24, controller-runtime v0.21, and uses manifestival for declarative resource management.

Key controllers:
- `OpenShiftBuildReconciler` -- reconciles the cluster-scoped `OpenShiftBuild` CR.
- `ShipwrightBuildReconciler` -- reconciles `ShipwrightBuild` resources.

## FIPS Compliance

All binaries are built with FIPS-validated cryptography:
- `GOEXPERIMENT=strictfipsruntime`
- `CGO_ENABLED=1`
- Build tags: `strictfipsruntime`
- Base images: `registry.redhat.io/ubi9/go-toolset` (build) and `ubi9-minimal` (runtime)

Do not introduce non-standard crypto libraries. Use only `crypto/tls` and standard library crypto packages.

## Disconnected Environment Support

- All container image references use SHA256 digests (`@sha256:...`), never floating tags.
- Images are injected via environment variables in `config/manager/manager.yaml`.
- No hardcoded external URLs. All registry references must be overridable.
- `USE_IMAGE_DIGESTS=true` is the default in the Makefile.

## Go Coding Standards

- **Error handling**: Wrap errors with context using `fmt.Errorf("operation: %w", err)`. Never silently discard errors.
- **Context propagation**: Always thread `context.Context` through the call stack. Every `Reconcile` method receives a context; pass it to all downstream calls.
- **Logging**: Use `logr` (via `log.FromContext(ctx)` or `ctrl.Log.WithName(...)`). The zap backend is configured in `cmd/main.go`. Do not use `klog` directly or `fmt.Print` for operational logging.
- **Vendoring**: Dependencies are vendored (`-mod vendor`). Run `go mod vendor` after dependency changes.

## Controller Patterns

- **Idempotent Reconcile**: `Reconcile` must be safe to call repeatedly with the same input. Use status conditions and finalizers to track state.
- **Manifestival**: Resources are applied via manifestival transforms. Transforms must be idempotent and must not assume prior state.
- **envtest / Ginkgo**: Controller tests use `envtest` for a real API server and Ginkgo v2 BDD syntax (`Describe`, `When`, `It`, `BeforeEach`).
- **RBAC markers**: Changes to `//+kubebuilder:rbac` comments affect generated ClusterRoles. Review these carefully.

## Container Security

- Runtime container runs as non-root: `USER 65532:65532`.
- Security context enforces `runAsNonRoot: true`, read-only root filesystem, and drops all capabilities.

## Downstream Maintenance

- When modifying code synced from upstream, mark changes with `// OCP DOWNSTREAM` comments explaining the reason.
- Never commit Secret manifests containing real credentials. Use placeholders.
- Every PR should include unit tests. Controller logic should include envtest or integration tests.
