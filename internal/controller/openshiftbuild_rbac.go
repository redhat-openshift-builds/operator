package controller

//+kubebuilder:rbac:groups=operator.openshift.io,resources=openshiftbuilds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.openshift.io,resources=openshiftbuilds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.openshift.io,resources=openshiftbuilds/finalizers,verbs=update

//+kubebuilder:rbac:groups=operator.shipwright.io,resources=shipwrightbuilds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.shipwright.io,resources=shipwrightbuilds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.shipwright.io,resources=shipwrightbuilds/finalizers,verbs=update
