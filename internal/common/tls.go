package common

import (
	"context"
	"fmt"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// openSSLToIANACiphers maps OpenSSL cipher suite names to IANA names.
// This conversion is necessary because OpenShift TLS profiles use OpenSSL
// cipher names (e.g., "ECDHE-RSA-AES128-GCM-SHA256") while Go's crypto/tls
// and webhook servers expect IANA names (e.g., "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256").
// Reference: https://www.iana.org/assignments/tls-parameters/tls-parameters.xml
var openSSLToIANACiphers = map[string]string{
	// TLS 1.3 ciphers (already in IANA format)
	"TLS_AES_128_GCM_SHA256":       "TLS_AES_128_GCM_SHA256",
	"TLS_AES_256_GCM_SHA384":       "TLS_AES_256_GCM_SHA384",
	"TLS_CHACHA20_POLY1305_SHA256": "TLS_CHACHA20_POLY1305_SHA256",

	// TLS 1.2 ciphers (OpenSSL to IANA conversion)
	"ECDHE-ECDSA-AES128-GCM-SHA256": "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
	"ECDHE-RSA-AES128-GCM-SHA256":   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	"ECDHE-ECDSA-AES256-GCM-SHA384": "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
	"ECDHE-RSA-AES256-GCM-SHA384":   "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	"ECDHE-ECDSA-CHACHA20-POLY1305": "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
	"ECDHE-RSA-CHACHA20-POLY1305":   "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
	"ECDHE-ECDSA-AES128-SHA256":     "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
	"ECDHE-RSA-AES128-SHA256":       "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
	"AES128-GCM-SHA256":             "TLS_RSA_WITH_AES_128_GCM_SHA256",
	"AES256-GCM-SHA384":             "TLS_RSA_WITH_AES_256_GCM_SHA384",
	"AES128-SHA256":                 "TLS_RSA_WITH_AES_128_CBC_SHA256",

	// TLS 1.0/1.1 ciphers (for Old profile)
	"ECDHE-ECDSA-AES128-SHA": "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
	"ECDHE-RSA-AES128-SHA":   "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	"ECDHE-ECDSA-AES256-SHA": "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	"ECDHE-RSA-AES256-SHA":   "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
	"AES128-SHA":             "TLS_RSA_WITH_AES_128_CBC_SHA",
	"AES256-SHA":             "TLS_RSA_WITH_AES_256_CBC_SHA",
	"DES-CBC3-SHA":           "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	"ECDHE-RSA-DES-CBC3-SHA": "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
}

// GetClusterTLSProfile reads the TLS security profile from the cluster APIServer resource.
// Returns the TLSSecurityProfile from spec.tlsSecurityProfile, or nil if not set.
// Returns nil if the component should not honor the cluster TLS profile based on tlsAdherence policy.
func GetClusterTLSProfile(ctx context.Context, cfg *rest.Config) (*configv1.TLSSecurityProfile, error) {
	logger := log.FromContext(ctx).WithName("tls")

	client, err := configclient.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create config client: %w", err)
	}

	apiserver, err := client.ConfigV1().APIServers().Get(ctx, OpenShiftBuildResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get APIServer CR: %w", err)
	}

	// Check if this component should honor the cluster TLS profile based on tlsAdherence policy.
	// The openshift-builds-operator is a "legacy adhering component" that opts into TLS compliance.
	if !shouldHonorClusterTLSProfile(apiserver.Spec.TLSAdherence) {
		logger.Info("skipping cluster TLS profile due to adherence policy",
			"adherence", apiserver.Spec.TLSAdherence,
			"note", "component will use Go defaults")
		return nil, nil
	}

	// Log the TLS profile being applied
	if apiserver.Spec.TLSSecurityProfile != nil {
		logger.Info("applying cluster TLS profile to operands",
			"type", apiserver.Spec.TLSSecurityProfile.Type,
			"adherence", apiserver.Spec.TLSAdherence)
	} else {
		logger.Info("no TLS profile configured in APIServer CR, using Intermediate defaults")
	}

	return apiserver.Spec.TLSSecurityProfile, nil
}

// shouldHonorClusterTLSProfile determines if the component should honor the cluster TLS profile
// based on the tlsAdherence policy.
//
// The openshift-builds-operator is considered a "legacy adhering component" - it has always
// honored the cluster TLS profile and continues to do so under both adherence policies:
//
//   - LegacyAdheringComponentsOnly (default): Components that already honor the TLS profile continue to do so.
//   - StrictAllComponents: All components must honor the TLS profile.
//
// This means the operator applies TLS settings in both modes.
func shouldHonorClusterTLSProfile(adherence configv1.TLSAdherencePolicy) bool {
	// When tlsAdherence is unset, it defaults to LegacyAdheringComponentsOnly behavior.
	// In both Legacy and Strict modes, legacy adhering components (like us) should honor the profile.
	switch adherence {
	case configv1.TLSAdherencePolicyNoOpinion, // Unset - defaults to Legacy
		configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly, // Explicit Legacy
		configv1.TLSAdherencePolicyStrictAllComponents:          // Strict
		return true
	default:
		// Unknown values should be treated as Strict per the epic guidance
		// "Components that encounter an unknown value for tlsAdherence should treat it
		// as StrictAllComponents and log a warning"
		return true
	}
}

// TLSProfileToFlags converts a TLSSecurityProfile to --tls-min-version and --tls-cipher-suites
// flag values.
// Returns empty strings if the profile is nil (use Go defaults).
func TLSProfileToFlags(profile *configv1.TLSSecurityProfile) (minVersion, cipherSuites string) {
	logger := log.Log.WithName("tls")

	if profile == nil {
		logger.V(1).Info("no TLS profile specified, defaulting to Intermediate profile")
		// Default to Intermediate profile
		profile = &configv1.TLSSecurityProfile{
			Type:         configv1.TLSProfileIntermediateType,
			Intermediate: &configv1.IntermediateTLSProfile{},
		}
	}

	var spec *configv1.TLSProfileSpec
	switch profile.Type {
	case configv1.TLSProfileCustomType:
		if profile.Custom != nil {
			spec = &configv1.TLSProfileSpec{
				MinTLSVersion: profile.Custom.MinTLSVersion,
				Ciphers:       profile.Custom.Ciphers,
			}
		}
	default:
		// Use predefined profile (Old, Intermediate, Modern)
		if profileSpec, ok := configv1.TLSProfiles[profile.Type]; ok {
			spec = profileSpec
		}
	}

	if spec == nil {
		return "", ""
	}

	minVersion = string(spec.MinTLSVersion) // e.g. "VersionTLS12"

	if len(spec.Ciphers) > 0 {
		// Convert OpenSSL cipher names to IANA format that operands expect.
		// OpenShift TLS profiles use mixed formats:
		//   - TLS 1.3 ciphers: "TLS_AES_128_GCM_SHA256" (already IANA)
		//   - TLS 1.2 ciphers: "ECDHE-RSA-AES128-GCM-SHA256" (OpenSSL format)
		// Operands (Shipwright, CSI driver) expect IANA format:
		//   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
		ianaCiphers := make([]string, 0, len(spec.Ciphers))
		skippedCiphers := []string{}
		for _, cipher := range spec.Ciphers {
			if ianaCipher, found := openSSLToIANACiphers[cipher]; found {
				ianaCiphers = append(ianaCiphers, ianaCipher)
			} else {
				// Track unsupported ciphers for logging
				skippedCiphers = append(skippedCiphers, cipher)
			}
		}
		if len(skippedCiphers) > 0 {
			logger.Info("unsupported ciphers in TLS profile will be skipped",
				"ciphers", strings.Join(skippedCiphers, ","),
				"note", "these cannot be negotiated by operand webhooks")
		}
		cipherSuites = strings.Join(ianaCiphers, ",")
		logger.V(1).Info("converted ciphers to IANA format",
			"count", len(ianaCiphers),
			"ciphers", cipherSuites)
	}

	return minVersion, cipherSuites
}

// ValidateTLSProfile validates a TLS security profile for correctness.
// Returns an error if the profile is invalid.
func ValidateTLSProfile(profile *configv1.TLSSecurityProfile) error {
	if profile == nil {
		return nil // nil is valid (uses defaults)
	}

	// Validate profile type
	validTypes := []configv1.TLSProfileType{
		configv1.TLSProfileOldType,
		configv1.TLSProfileIntermediateType,
		configv1.TLSProfileModernType,
		configv1.TLSProfileCustomType,
	}
	typeValid := false
	for _, validType := range validTypes {
		if profile.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return fmt.Errorf("invalid TLS profile type: %s", profile.Type)
	}

	// For Custom profiles, validate the spec
	if profile.Type == configv1.TLSProfileCustomType {
		if profile.Custom == nil {
			return fmt.Errorf("custom profile type requires custom spec")
		}

		spec := &profile.Custom.TLSProfileSpec

		// Validate MinTLSVersion
		validVersions := []configv1.TLSProtocolVersion{
			configv1.VersionTLS10,
			configv1.VersionTLS11,
			configv1.VersionTLS12,
			configv1.VersionTLS13,
		}
		versionValid := false
		for _, validVersion := range validVersions {
			if spec.MinTLSVersion == validVersion {
				versionValid = true
				break
			}
		}
		if !versionValid {
			return fmt.Errorf("invalid MinTLSVersion in custom profile: %s", spec.MinTLSVersion)
		}

		// Validate that at least one cipher is provided
		if len(spec.Ciphers) == 0 {
			return fmt.Errorf("custom profile must specify at least one cipher")
		}
	}

	return nil
}
