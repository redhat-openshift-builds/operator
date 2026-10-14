/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	shipwrightWebhookName = "shipwright-build-webhook"
	csiWebhookName        = "shared-resource-csi-driver-webhook"
	apiServerName         = "cluster"
	reconcileTimeout      = 2 * time.Minute
	reconcilePollInterval = 5 * time.Second
)

// TestTLSProfileModern verifies Modern profile (TLS 1.3) is applied correctly
func TestTLSProfileModern(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	kubeClient, configClient := setupClients(t)

	// Save original TLS profile for cleanup
	originalProfile, originalAdherence := getAPIServerTLSConfig(t, ctx, configClient)
	defer restoreTLSProfile(t, ctx, configClient, originalProfile, originalAdherence)

	// Set Modern TLS profile
	t.Log("Setting Modern TLS profile on APIServer CR")
	modernProfile := &configv1.TLSSecurityProfile{
		Type:   configv1.TLSProfileModernType,
		Modern: &configv1.ModernTLSProfile{},
	}
	setAPIServerTLSProfile(t, ctx, configClient, modernProfile, configv1.TLSAdherencePolicyStrictAllComponents)

	// Wait for operator to reconcile
	t.Log("Waiting for operator reconciliation...")
	waitForReconciliation(t, ctx, kubeClient, "Modern")

	// Verify Shipwright webhook has correct TLS flags
	t.Log("Verifying Shipwright webhook TLS flags")
	minVersion, ciphers := getDeploymentTLSFlags(t, ctx, kubeClient, openshiftBuildsNamespace, shipwrightWebhookName, shipwrightWebhookName)
	if minVersion != "VersionTLS13" {
		t.Errorf("Expected Shipwright webhook minVersion=VersionTLS13, got %s", minVersion)
	}
	if !strings.Contains(ciphers, "TLS_AES_128_GCM_SHA256") {
		t.Errorf("Expected Shipwright webhook to have TLS 1.3 ciphers, got %s", ciphers)
	}
	if strings.Contains(ciphers, "TLS_ECDHE") {
		t.Errorf("Modern profile should not have TLS 1.2 ciphers, got %s", ciphers)
	}

	// Verify CSI webhook has correct TLS flags
	t.Log("Verifying CSI webhook TLS flags")
	minVersionCSI, ciphersCSI := getDeploymentTLSFlags(t, ctx, kubeClient, openshiftBuildsNamespace, csiWebhookName, csiWebhookName)
	if minVersionCSI != minVersion {
		t.Errorf("CSI webhook minVersion mismatch: expected %s, got %s", minVersion, minVersionCSI)
	}
	if ciphersCSI != ciphers {
		t.Errorf("CSI webhook ciphers mismatch: expected %s, got %s", ciphers, ciphersCSI)
	}

	t.Log("Modern TLS profile successfully applied to all webhooks")
}

// TestTLSProfileIntermediate verifies Intermediate profile (TLS 1.2+) is applied
func TestTLSProfileIntermediate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	kubeClient, configClient := setupClients(t)

	originalProfile, originalAdherence := getAPIServerTLSConfig(t, ctx, configClient)
	defer restoreTLSProfile(t, ctx, configClient, originalProfile, originalAdherence)

	t.Log("Setting Intermediate TLS profile")
	intermediateProfile := &configv1.TLSSecurityProfile{
		Type:         configv1.TLSProfileIntermediateType,
		Intermediate: &configv1.IntermediateTLSProfile{},
	}
	setAPIServerTLSProfile(t, ctx, configClient, intermediateProfile, configv1.TLSAdherencePolicyStrictAllComponents)

	waitForReconciliation(t, ctx, kubeClient, "Intermediate")

	minVersion, ciphers := getDeploymentTLSFlags(t, ctx, kubeClient, openshiftBuildsNamespace, shipwrightWebhookName, shipwrightWebhookName)
	if minVersion != "VersionTLS12" {
		t.Errorf("Expected minVersion=VersionTLS12, got %s", minVersion)
	}
	if !strings.Contains(ciphers, "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256") {
		t.Errorf("Expected Intermediate ciphers, got %s", ciphers)
	}

	t.Log("Intermediate TLS profile successfully applied")
}

// TestTLSProfileOld verifies Old profile (TLS 1.0+) for legacy clients
func TestTLSProfileOld(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	kubeClient, configClient := setupClients(t)

	originalProfile, originalAdherence := getAPIServerTLSConfig(t, ctx, configClient)
	defer restoreTLSProfile(t, ctx, configClient, originalProfile, originalAdherence)

	t.Log("Setting Old TLS profile")
	oldProfile := &configv1.TLSSecurityProfile{
		Type: configv1.TLSProfileOldType,
		Old:  &configv1.OldTLSProfile{},
	}
	setAPIServerTLSProfile(t, ctx, configClient, oldProfile, configv1.TLSAdherencePolicyStrictAllComponents)

	waitForReconciliation(t, ctx, kubeClient, "Old")

	minVersion, ciphers := getDeploymentTLSFlags(t, ctx, kubeClient, openshiftBuildsNamespace, shipwrightWebhookName, shipwrightWebhookName)
	if minVersion != "VersionTLS10" {
		t.Errorf("Expected minVersion=VersionTLS10, got %s", minVersion)
	}
	if !strings.Contains(ciphers, "TLS_RSA_WITH_3DES_EDE_CBC_SHA") {
		t.Errorf("Expected Old profile ciphers including 3DES, got %s", ciphers)
	}

	t.Log("Old TLS profile successfully applied")
}

// TestTLSProfileCustomMinVersion verifies custom minimum TLS version
func TestTLSProfileCustomMinVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	kubeClient, configClient := setupClients(t)

	originalProfile, originalAdherence := getAPIServerTLSConfig(t, ctx, configClient)
	defer restoreTLSProfile(t, ctx, configClient, originalProfile, originalAdherence)

	t.Log("Setting Custom TLS profile with TLS 1.3 minimum")
	customProfile := &configv1.TLSSecurityProfile{
		Type: configv1.TLSProfileCustomType,
		Custom: &configv1.CustomTLSProfile{
			TLSProfileSpec: configv1.TLSProfileSpec{
				MinTLSVersion: configv1.VersionTLS13,
				Ciphers: []string{
					"TLS_AES_128_GCM_SHA256",
					"TLS_AES_256_GCM_SHA384",
				},
			},
		},
	}
	setAPIServerTLSProfile(t, ctx, configClient, customProfile, configv1.TLSAdherencePolicyStrictAllComponents)

	waitForReconciliation(t, ctx, kubeClient, "Custom")

	minVersion, ciphers := getDeploymentTLSFlags(t, ctx, kubeClient, openshiftBuildsNamespace, shipwrightWebhookName, shipwrightWebhookName)
	if minVersion != "VersionTLS13" {
		t.Errorf("Expected custom minVersion=VersionTLS13, got %s", minVersion)
	}
	if !strings.Contains(ciphers, "TLS_AES_128_GCM_SHA256") || !strings.Contains(ciphers, "TLS_AES_256_GCM_SHA384") {
		t.Errorf("Expected custom ciphers, got %s", ciphers)
	}

	t.Log("Custom TLS profile successfully applied")
}

// TestTLSProfileCustomCiphers verifies custom cipher list is applied
func TestTLSProfileCustomCiphers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	kubeClient, configClient := setupClients(t)

	originalProfile, originalAdherence := getAPIServerTLSConfig(t, ctx, configClient)
	defer restoreTLSProfile(t, ctx, configClient, originalProfile, originalAdherence)

	t.Log("Setting Custom TLS profile with specific ciphers")
	customProfile := &configv1.TLSSecurityProfile{
		Type: configv1.TLSProfileCustomType,
		Custom: &configv1.CustomTLSProfile{
			TLSProfileSpec: configv1.TLSProfileSpec{
				MinTLSVersion: configv1.VersionTLS12,
				Ciphers: []string{
					"ECDHE-RSA-AES128-GCM-SHA256", // OpenSSL format - will convert to IANA
					"TLS_AES_128_GCM_SHA256",
					"UNKNOWN_CIPHER", // Should be skipped
				},
			},
		},
	}
	setAPIServerTLSProfile(t, ctx, configClient, customProfile, configv1.TLSAdherencePolicyStrictAllComponents)

	waitForReconciliation(t, ctx, kubeClient, "Custom")

	_, ciphers := getDeploymentTLSFlags(t, ctx, kubeClient, openshiftBuildsNamespace, shipwrightWebhookName, shipwrightWebhookName)
	if !strings.Contains(ciphers, "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256") {
		t.Errorf("Expected converted IANA cipher name, got %s", ciphers)
	}
	if !strings.Contains(ciphers, "TLS_AES_128_GCM_SHA256") {
		t.Errorf("Expected TLS 1.3 cipher, got %s", ciphers)
	}
	if strings.Contains(ciphers, "UNKNOWN_CIPHER") {
		t.Errorf("Unsupported cipher should be skipped, got %s", ciphers)
	}

	t.Log("Custom cipher list successfully applied with unsupported ciphers skipped")
}

// TestTLSAdherenceStrict verifies strict mode enforcement
func TestTLSAdherenceStrict(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	kubeClient, configClient := setupClients(t)

	originalProfile, originalAdherence := getAPIServerTLSConfig(t, ctx, configClient)
	defer restoreTLSProfile(t, ctx, configClient, originalProfile, originalAdherence)

	t.Log("Setting TLSAdherence to StrictAllComponents with Modern profile")
	modernProfile := &configv1.TLSSecurityProfile{
		Type:   configv1.TLSProfileModernType,
		Modern: &configv1.ModernTLSProfile{},
	}
	setAPIServerTLSProfile(t, ctx, configClient, modernProfile, configv1.TLSAdherencePolicyStrictAllComponents)

	waitForReconciliation(t, ctx, kubeClient, "Modern")

	// Verify operator applied TLS config (operator is legacy adhering component)
	minVersion, _ := getDeploymentTLSFlags(t, ctx, kubeClient, openshiftBuildsNamespace, shipwrightWebhookName, shipwrightWebhookName)
	if minVersion != "VersionTLS13" {
		t.Errorf("Strict adherence mode should enforce TLS profile on operator, got minVersion=%s", minVersion)
	}

	t.Log("Strict adherence mode successfully enforced")
}

// TestTLSAdherenceLegacy verifies legacy mode allows opt-in
func TestTLSAdherenceLegacy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	kubeClient, configClient := setupClients(t)

	originalProfile, originalAdherence := getAPIServerTLSConfig(t, ctx, configClient)
	defer restoreTLSProfile(t, ctx, configClient, originalProfile, originalAdherence)

	t.Log("Setting TLSAdherence to LegacyAdheringComponentsOnly with Intermediate profile")
	intermediateProfile := &configv1.TLSSecurityProfile{
		Type:         configv1.TLSProfileIntermediateType,
		Intermediate: &configv1.IntermediateTLSProfile{},
	}
	setAPIServerTLSProfile(t, ctx, configClient, intermediateProfile, configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly)

	waitForReconciliation(t, ctx, kubeClient, "Intermediate")

	// Verify operator (as legacy component) still applies TLS config
	minVersion, _ := getDeploymentTLSFlags(t, ctx, kubeClient, openshiftBuildsNamespace, shipwrightWebhookName, shipwrightWebhookName)
	if minVersion != "VersionTLS12" {
		t.Errorf("Legacy adherence mode should still honor TLS for legacy components, got minVersion=%s", minVersion)
	}

	t.Log("Legacy adherence mode successfully applied to legacy component")
}

// TestTLSProfileNil verifies default Intermediate profile is used when no profile set
func TestTLSProfileNil(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	kubeClient, configClient := setupClients(t)

	originalProfile, originalAdherence := getAPIServerTLSConfig(t, ctx, configClient)
	defer restoreTLSProfile(t, ctx, configClient, originalProfile, originalAdherence)

	t.Log("Removing TLS profile from APIServer CR (should default to Intermediate)")
	setAPIServerTLSProfile(t, ctx, configClient, nil, configv1.TLSAdherencePolicyStrictAllComponents)

	waitForReconciliation(t, ctx, kubeClient, "Intermediate")

	// Verify Intermediate defaults are applied
	minVersion, ciphers := getDeploymentTLSFlags(t, ctx, kubeClient, openshiftBuildsNamespace, shipwrightWebhookName, shipwrightWebhookName)
	if minVersion != "VersionTLS12" {
		t.Errorf("Expected default Intermediate minVersion=VersionTLS12, got %s", minVersion)
	}
	if !strings.Contains(ciphers, "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256") {
		t.Errorf("Expected default Intermediate ciphers, got %s", ciphers)
	}

	t.Log("Default Intermediate profile successfully applied")
}

// setupClients creates Kubernetes and OpenShift config clients from kubeconfig
func setupClients(t *testing.T) (*kubernetes.Clientset, *configclient.Clientset) {
	t.Helper()

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		t.Fatalf("Failed to load kubeconfig: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	configClient, err := configclient.NewForConfig(config)
	if err != nil {
		t.Fatalf("Failed to create OpenShift config client: %v", err)
	}

	return kubeClient, configClient
}

// getAPIServerTLSConfig gets current TLS profile and adherence from APIServer CR
func getAPIServerTLSConfig(t *testing.T, ctx context.Context, configClient *configclient.Clientset) (*configv1.TLSSecurityProfile, configv1.TLSAdherencePolicy) {
	t.Helper()

	apiserver, err := configClient.ConfigV1().APIServers().Get(ctx, apiServerName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get APIServer CR: %v", err)
	}

	return apiserver.Spec.TLSSecurityProfile, apiserver.Spec.TLSAdherence
}

// setAPIServerTLSProfile updates the APIServer CR with new TLS profile and adherence
func setAPIServerTLSProfile(t *testing.T, ctx context.Context, configClient *configclient.Clientset, profile *configv1.TLSSecurityProfile, adherence configv1.TLSAdherencePolicy) {
	t.Helper()

	apiserver, err := configClient.ConfigV1().APIServers().Get(ctx, apiServerName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get APIServer CR: %v", err)
	}

	apiserver.Spec.TLSSecurityProfile = profile
	apiserver.Spec.TLSAdherence = adherence

	_, err = configClient.ConfigV1().APIServers().Update(ctx, apiserver, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update APIServer CR: %v", err)
	}

	t.Logf("Updated APIServer CR with TLS profile type=%v, adherence=%v", getProfileType(profile), adherence)
}

// getDeploymentTLSFlags extracts TLS flags from a deployment's container args
func getDeploymentTLSFlags(t *testing.T, ctx context.Context, kubeClient *kubernetes.Clientset, namespace, deploymentName, containerName string) (minVersion, ciphers string) {
	t.Helper()

	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get deployment %s/%s: %v", namespace, deploymentName, err)
	}

	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			for _, arg := range container.Args {
				if strings.HasPrefix(arg, "--tls-min-version=") {
					minVersion = strings.TrimPrefix(arg, "--tls-min-version=")
				}
				if strings.HasPrefix(arg, "--tls-cipher-suites=") {
					ciphers = strings.TrimPrefix(arg, "--tls-cipher-suites=")
				}
			}
			return minVersion, ciphers
		}
	}

	t.Fatalf("Container %s not found in deployment %s/%s", containerName, namespace, deploymentName)
	return "", ""
}

// waitForReconciliation waits for operator to reconcile and deployment to update
func waitForReconciliation(t *testing.T, ctx context.Context, kubeClient *kubernetes.Clientset, expectedProfileType string) {
	t.Helper()

	err := wait.PollUntilContextTimeout(ctx, reconcilePollInterval, reconcileTimeout, true, func(ctx context.Context) (bool, error) {
		// Check if Shipwright webhook deployment has been updated
		deployment, err := kubeClient.AppsV1().Deployments(openshiftBuildsNamespace).Get(ctx, shipwrightWebhookName, metav1.GetOptions{})
		if err != nil {
			t.Logf("Waiting for deployment %s: %v", shipwrightWebhookName, err)
			return false, nil
		}

		// Check if deployment has TLS args (indicates reconciliation happened)
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == shipwrightWebhookName {
				for _, arg := range container.Args {
					if strings.HasPrefix(arg, "--tls-min-version=") {
						t.Logf("Reconciliation complete - TLS flags detected in webhook")
						return true, nil
					}
				}
			}
		}

		t.Logf("Waiting for TLS flags to appear in webhook deployment...")
		return false, nil
	})

	if err != nil {
		t.Fatalf("Timeout waiting for reconciliation: %v", err)
	}
}

// restoreTLSProfile restores the original APIServer TLS configuration
func restoreTLSProfile(t *testing.T, ctx context.Context, configClient *configclient.Clientset, original *configv1.TLSSecurityProfile, originalAdherence configv1.TLSAdherencePolicy) {
	t.Helper()

	t.Log("Restoring original TLS profile")
	setAPIServerTLSProfile(t, ctx, configClient, original, originalAdherence)
}

// getProfileType safely gets the TLS profile type name
func getProfileType(profile *configv1.TLSSecurityProfile) string {
	if profile == nil {
		return "nil (defaults to Intermediate)"
	}
	return string(profile.Type)
}
