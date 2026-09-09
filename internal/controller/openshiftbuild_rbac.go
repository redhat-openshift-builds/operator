package controller

//+kubebuilder:rbac:groups=operator.openshift.io,resources=openshiftbuilds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.openshift.io,resources=openshiftbuilds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.openshift.io,resources=openshiftbuilds/finalizers,verbs=update
//+kubebuilder:rbac:groups=operator.shipwright.io,resources=shipwrightbuilds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.shipwright.io,resources=shipwrightbuilds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.shipwright.io,resources=shipwrightbuilds/finalizers,verbs=update
//+kubebuilder:rbac:groups=storage.k8s.io,resources=csidrivers,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=storage.k8s.io,resources=csidrivers,resourceNames=csi.sharedresource.openshift.io,verbs=update;patch;delete
//+kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,resourceNames=csi-driver-shared-resource,verbs=get;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,resourceNames=csi-driver-shared-resource;openshift-builds-operator,verbs=get;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=daemonsets,namespace=openshift-builds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,namespace=openshift-builds,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=apps,resources=deployments,namespace=openshift-builds,resourceNames=shared-resource-csi-driver-webhook,verbs=update;patch;delete
//+kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,namespace=openshift-builds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,namespace=openshift-builds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,namespace=openshift-builds,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,namespace=openshift-builds,resourceNames=csi-driver-shared-resource,verbs=update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,namespace=openshift-builds,resourceNames=shared-resource-csi-driver-webhook,verbs=update;patch;delete
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;create
//+kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,resourceNames=validation.webhook.csidriversharedresource;pod.csi.sharedresource.openshift.io,verbs=update;patch;delete
//+kubebuilder:rbac:groups=sharedresource.openshift.io,resources=sharedconfigmaps;sharedsecrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,namespace=openshift-builds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,resourceNames=sharedconfigmaps.sharedresource.openshift.io;sharedsecrets.sharedresource.openshift.io,verbs=get;list;watch;create;update;delete;patch
