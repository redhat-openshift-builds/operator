package common

import (
	"context"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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
func GetClusterTLSProfile(ctx context.Context, cfg *rest.Config) (*configv1.TLSSecurityProfile, error) {
	client, err := configclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	apiserver, err := client.ConfigV1().APIServers().Get(ctx, OpenShiftBuildResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return apiserver.Spec.TLSSecurityProfile, nil
}

// TLSProfileToFlags converts a TLSSecurityProfile to --tls-min-version and --tls-cipher-suites flag values.
// Returns empty strings if the profile is nil (use Go defaults).
func TLSProfileToFlags(profile *configv1.TLSSecurityProfile) (minVersion string, cipherSuites string) {
	if profile == nil {
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
		for _, cipher := range spec.Ciphers {
			if ianaCipher, found := openSSLToIANACiphers[cipher]; found {
				ianaCiphers = append(ianaCiphers, ianaCipher)
			}
			// Silently skip unsupported ciphers (they cannot be negotiated anyway)
		}
		cipherSuites = strings.Join(ianaCiphers, ",")
	}

	return minVersion, cipherSuites
}
