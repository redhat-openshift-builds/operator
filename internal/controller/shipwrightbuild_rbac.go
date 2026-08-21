package controller

// To minimize the risk of privilege escalation or destructive behavior, the controller is only
// allowed to modify named resources that deploy Shipwright Build. This is especially true for the
// cluster roles and custom resource definitions included in the release manifest.
//
// Namespaced operand resources are scoped to the operator's install namespace (openshift-builds):
// every operand manifest is namespace-injected to that namespace before it is applied, so the
// controller never writes namespaced resources anywhere else.

// Namespaced workloads (openshift-builds): create broadly, mutate only the named Deployments.
// +kubebuilder:rbac:groups=apps,resources=deployments,namespace=openshift-builds,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=apps,resources=deployments,namespace=openshift-builds,resourceNames=shipwright-build-controller,verbs=update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers,namespace=openshift-builds,resourceNames=shipwright-build-controller,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,namespace=openshift-builds,resourceNames=shipwright-build-webhook,verbs=update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers,namespace=openshift-builds,resourceNames=shipwright-build-webhook,verbs=update

// Namespaces are cluster-scoped; the controller only reads them and creates the target namespace if missing.
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create

// Namespaced core resources (openshift-builds). Secrets are read-only; pods, events and limitranges
// are never written by the controller (events for leader election are covered by the leader-election Role).
// +kubebuilder:rbac:groups=core,resources=configmaps;services,namespace=openshift-builds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,namespace=openshift-builds,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,namespace=openshift-builds,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,namespace=openshift-builds,resourceNames=shipwright-build-controller,verbs=update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,namespace=openshift-builds,resourceNames=shipwright-build-webhook,verbs=update;patch;delete

// Cluster-scoped CRDs: create broadly, mutate only the Shipwright CRDs.
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,resourceNames=builds.shipwright.io;buildruns.shipwright.io;buildstrategies.shipwright.io;clusterbuildstrategies.shipwright.io,verbs=update;patch;delete

// Cluster-scoped RBAC: create broadly (also covers the shared-resource CSI cluster RBAC objects),
// mutate only the named ClusterRoles/ClusterRoleBindings.
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,resourceNames=shipwright-build-aggregate-edit;shipwright-build-aggregate-view;shipwright-build-controller,verbs=update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,resourceNames=shipwright-build-controller,verbs=update;patch;delete

// Namespaced RBAC (openshift-builds): create broadly, mutate only the named Role/RoleBinding.
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,namespace=openshift-builds,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,namespace=openshift-builds,resourceNames=shipwright-build-controller,verbs=update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,namespace=openshift-builds,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,namespace=openshift-builds,resourceNames=shipwright-build-controller,verbs=update;patch;delete

// Cluster-scoped operands and the ShipwrightBuild CR itself.
// +kubebuilder:rbac:groups=shipwright.io,resources=clusterbuildstrategies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.shipwright.io,resources=shipwrightbuilds,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=operator.shipwright.io,resources=shipwrightbuilds/finalizers,verbs=update
// +kubebuilder:rbac:groups=operator.shipwright.io,resources=shipwrightbuilds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.tekton.dev,resources=tektonconfigs,verbs=get;list;create
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io/v1beta1,resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete

// cert-manager objects are namespaced; create broadly, mutate only the named Issuer/Certificate.
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,namespace=openshift-builds,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,namespace=openshift-builds,resourceNames=selfsigned-issuer,verbs=update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,namespace=openshift-builds,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,namespace=openshift-builds,resourceNames=shipwright-build-webhook-cert,verbs=update;patch;delete
