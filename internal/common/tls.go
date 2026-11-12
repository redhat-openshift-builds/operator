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
	"ECDHE-ECDSA-AES128-SHA":    "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
	"ECDHE-RSA-AES128-SHA":      "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
	"ECDHE-ECDSA-AES256-SHA384": "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384",
	"ECDHE-RSA-AES256-SHA384":   "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384",
	"ECDHE-ECDSA-AES256-SHA":    "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
	"ECDHE-RSA-AES256-SHA":      "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
	"AES128-SHA":                "TLS_RSA_WITH_AES_128_CBC_SHA",
	"AES256-SHA256":             "TLS_RSA_WITH_AES_256_CBC_SHA256",
	"AES256-SHA":                "TLS_RSA_WITH_AES_256_CBC_SHA",
	"DES-CBC3-SHA":              "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
	"ECDHE-RSA-DES-CBC3-SHA":    "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
}

// openSSLCurveToIANA maps OpenSSL elliptic curve names to IANA names.
var openSSLCurveToIANA = map[string]string{
	// Post-Quantum Cryptography (PQC) hybrid curves
	"X25519MLKEM768":     "X25519MLKEM768",
	"SecP256r1MLKEM768":  "SecP256r1MLKEM768",
	"SecP384r1MLKEM1024": "SecP384r1MLKEM1024",

	// Traditional curves - OpenSSL to IANA format
	"X25519":     "X25519",
	"prime256v1": "secp256r1",
	"secp384r1":  "secp384r1",
	"secp521r1":  "secp521r1",
	"P-256":      "secp256r1",
	"P-384":      "secp384r1",
	"P-521":      "secp521r1",
	"x25519":     "X25519",
	"secp256r1":  "secp256r1",
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

// ConvertOpenSSLCurveToIANA converts an OpenSSL curve name to IANA format
func ConvertOpenSSLCurveToIANA(openSSLCurve string) (string, bool) {
	ianaCurve, found := openSSLCurveToIANA[openSSLCurve]
	return ianaCurve, found
}

// TLSProfileToFlags converts a TLSSecurityProfile to --tls-min-version, --tls-cipher-suites,
// and --tls-curve-preferences flag values.
// Returns empty strings if the profile is nil (use Go defaults).
func TLSProfileToFlags(profile *configv1.TLSSecurityProfile) (minVersion, cipherSuites, curves string) {
	logger := log.Log.WithName("tls")

	if profile == nil {
		return "", "", ""
	}

	var spec *configv1.TLSProfileSpec
	switch profile.Type {
	case configv1.TLSProfileCustomType:
		if profile.Custom != nil {
			spec = &configv1.TLSProfileSpec{
				MinTLSVersion: profile.Custom.MinTLSVersion,
				Ciphers:       profile.Custom.Ciphers,
			}
			// Only include Groups if explicitly set in Custom profile
			if len(profile.Custom.Groups) > 0 {
				spec.Groups = profile.Custom.Groups
			}
		}
	default:
		// Use predefined profile (Old, Intermediate, Modern)
		if profileSpec, ok := configv1.TLSProfiles[profile.Type]; ok {
			spec = profileSpec
		}
	}

	if spec == nil {
		return "", "", ""
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

	// Convert elliptic curves from OpenSSL to IANA format
	// Preserve the order from the profile (PQC curves first, then traditional)
	if len(spec.Groups) > 0 {
		ianaCurves := make([]string, 0, len(spec.Groups))
		skippedCurves := []string{}
		for _, openSSLCurve := range spec.Groups {
			curveStr := string(openSSLCurve)
			if ianaCurve, found := ConvertOpenSSLCurveToIANA(curveStr); found {
				ianaCurves = append(ianaCurves, ianaCurve)
			} else {
				skippedCurves = append(skippedCurves, curveStr)
			}
		}
		if len(skippedCurves) > 0 {
			logger.Info("unsupported curves in TLS profile will be skipped",
				"curves", strings.Join(skippedCurves, ","))
		}
		curves = strings.Join(ianaCurves, ",")
	}

	return minVersion, cipherSuites, curves
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

// ParseTLSVersion converts a TLS version string to crypto/tls constant.
// Used to configure the operator's metrics server TLS settings from cluster TLS profile.
func ParseTLSVersion(version string) uint16 {
	switch version {
	case "VersionTLS10":
		return 0x0301 // tls.VersionTLS10
	case "VersionTLS11":
		return 0x0302 // tls.VersionTLS11
	case "VersionTLS12":
		return 0x0303 // tls.VersionTLS12
	case "VersionTLS13":
		return 0x0304 // tls.VersionTLS13
	default:
		return 0 // Use Go default
	}
}

// ParseCipherSuites converts comma-separated IANA cipher names to crypto/tls cipher suite IDs.
// Used to configure the operator's metrics server TLS settings from cluster TLS profile.
func ParseCipherSuites(cipherSuites string) []uint16 {
	if cipherSuites == "" {
		return nil
	}

	// Map of IANA cipher names to crypto/tls constants
	cipherMap := map[string]uint16{
		// TLS 1.3 ciphers
		"TLS_AES_128_GCM_SHA256":       0x1301,
		"TLS_AES_256_GCM_SHA384":       0x1302,
		"TLS_CHACHA20_POLY1305_SHA256": 0x1303,

		// TLS 1.2 ECDHE ciphers
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       0xc02b,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         0xc02f,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       0xc02c,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         0xc030,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": 0xcca9,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   0xcca8,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256":       0xc023,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":         0xc027,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":          0xc009,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":            0xc013,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384":       0xc024,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384":         0xc028,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":          0xc00a,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":            0xc014,

		// TLS 1.2 RSA ciphers
		"TLS_RSA_WITH_AES_128_GCM_SHA256": 0x009c,
		"TLS_RSA_WITH_AES_256_GCM_SHA384": 0x009d,
		"TLS_RSA_WITH_AES_128_CBC_SHA256": 0x003c,
		"TLS_RSA_WITH_AES_256_CBC_SHA256": 0x003d,
		"TLS_RSA_WITH_AES_128_CBC_SHA":    0x002f,
		"TLS_RSA_WITH_AES_256_CBC_SHA":    0x0035,
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA":   0x000a,

		// DHE ciphers
		"TLS_DHE_RSA_WITH_AES_128_GCM_SHA256": 0x009e,
		"TLS_DHE_RSA_WITH_AES_256_GCM_SHA384": 0x009f,
		"TLS_DHE_RSA_WITH_AES_128_CBC_SHA256": 0x0067,
		"TLS_DHE_RSA_WITH_AES_256_CBC_SHA256": 0x006b,
		"TLS_DHE_RSA_WITH_AES_128_CBC_SHA":    0x0033,
		"TLS_DHE_RSA_WITH_AES_256_CBC_SHA":    0x0039,
	}

	cipherNames := strings.Split(cipherSuites, ",")
	var cipherIDs []uint16
	for _, name := range cipherNames {
		name = strings.TrimSpace(name)
		if id, ok := cipherMap[name]; ok {
			cipherIDs = append(cipherIDs, id)
		}
	}

	return cipherIDs
}

// ParseCurvePreferences converts comma-separated IANA curve names to crypto/tls curve IDs.
// Used to configure the operator's metrics server TLS settings from cluster TLS profile.
func ParseCurvePreferences(curves string) []uint16 {
	if curves == "" {
		return nil
	}

	// Map of IANA curve names to crypto/tls CurveID constants
	// CurveID is uint16 under the hood: type CurveID uint16
	// Only includes curves supported by OpenShift TLSGroup API
	// Accepts both lowercase (IANA standard) and capitalized (OpenShift API) variants
	curveMap := map[string]uint16{
		"X25519":    0x001d, // X25519 (CurveID 29)
		"x25519":    0x001d, // x25519 lowercase variant
		"secp256r1": 0x0017, // P-256 (CurveID 23)
		"secp384r1": 0x0018, // P-384 (CurveID 24)
		"secp521r1": 0x0019, // P-521 (CurveID 25)
	}

	curveNames := strings.Split(curves, ",")
	var curveIDs []uint16
	for _, name := range curveNames {
		name = strings.TrimSpace(name)
		if id, ok := curveMap[name]; ok {
			curveIDs = append(curveIDs, id)
		}
	}

	return curveIDs
}
