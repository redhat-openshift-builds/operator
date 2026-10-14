package common

import (
	"strings"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
)

func TestTLSProfileToFlags(t *testing.T) {
	tests := []struct {
		name               string
		profile            *configv1.TLSSecurityProfile
		expectedMinVersion string
		expectedCiphers    string
	}{
		{
			name:               "nil profile defaults to Intermediate",
			profile:            nil,
			expectedMinVersion: "VersionTLS12",
			// OpenSSL format converted to IANA format by crypto.OpenSSLToIANACipherSuites
			expectedCiphers: "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
		},
		{
			name: "Intermediate profile",
			profile: &configv1.TLSSecurityProfile{
				Type:         configv1.TLSProfileIntermediateType,
				Intermediate: &configv1.IntermediateTLSProfile{},
			},
			expectedMinVersion: "VersionTLS12",
			// OpenSSL format converted to IANA format by crypto.OpenSSLToIANACipherSuites
			expectedCiphers: "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
		},
		{
			name: "Modern profile",
			profile: &configv1.TLSSecurityProfile{
				Type:   configv1.TLSProfileModernType,
				Modern: &configv1.ModernTLSProfile{},
			},
			expectedMinVersion: "VersionTLS13",
			expectedCiphers:    "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256",
		},
		{
			name: "Old profile",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileOldType,
				Old:  &configv1.OldTLSProfile{},
			},
			expectedMinVersion: "VersionTLS10",
			// OpenSSL format converted to IANA format by crypto.OpenSSLToIANACipherSuites
			expectedCiphers: "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,TLS_RSA_WITH_AES_128_GCM_SHA256,TLS_RSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_128_CBC_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA,TLS_RSA_WITH_AES_256_CBC_SHA,TLS_RSA_WITH_3DES_EDE_CBC_SHA",
		},
		{
			name: "Custom profile with TLS 1.2",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						MinTLSVersion: configv1.VersionTLS12,
						Ciphers: []string{
							"TLS_AES_128_GCM_SHA256",
							"TLS_AES_256_GCM_SHA384",
						},
					},
				},
			},
			expectedMinVersion: "VersionTLS12",
			expectedCiphers:    "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384",
		},
		{
			name: "Custom profile with TLS 1.3",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						MinTLSVersion: configv1.VersionTLS13,
						Ciphers:       []string{"TLS_AES_128_GCM_SHA256"},
					},
				},
			},
			expectedMinVersion: "VersionTLS13",
			expectedCiphers:    "TLS_AES_128_GCM_SHA256",
		},
		{
			name: "Custom profile with no ciphers",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						MinTLSVersion: configv1.VersionTLS12,
						Ciphers:       []string{},
					},
				},
			},
			expectedMinVersion: "VersionTLS12",
			expectedCiphers:    "",
		},
		{
			name: "Custom profile is nil",
			profile: &configv1.TLSSecurityProfile{
				Type:   configv1.TLSProfileCustomType,
				Custom: nil,
			},
			expectedMinVersion: "",
			expectedCiphers:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minVersion, cipherSuites := TLSProfileToFlags(tt.profile)
			if minVersion != tt.expectedMinVersion {
				t.Errorf("TLSProfileToFlags() minVersion = %v, want %v", minVersion, tt.expectedMinVersion)
			}
			if cipherSuites != tt.expectedCiphers {
				t.Errorf("TLSProfileToFlags() cipherSuites = %v, want %v", cipherSuites, tt.expectedCiphers)
			}
		})
	}
}

func TestShouldHonorClusterTLSProfile(t *testing.T) {
	tests := []struct {
		name        string
		adherence   configv1.TLSAdherencePolicy
		shouldHonor bool
	}{
		{
			name:        "NoOpinion (unset) - should honor",
			adherence:   configv1.TLSAdherencePolicyNoOpinion,
			shouldHonor: true,
		},
		{
			name:        "LegacyAdheringComponentsOnly - should honor",
			adherence:   configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly,
			shouldHonor: true,
		},
		{
			name:        "StrictAllComponents - should honor",
			adherence:   configv1.TLSAdherencePolicyStrictAllComponents,
			shouldHonor: true,
		},
		{
			name:        "Unknown value - should honor (treat as Strict)",
			adherence:   "UnknownPolicy",
			shouldHonor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldHonorClusterTLSProfile(tt.adherence)
			if result != tt.shouldHonor {
				t.Errorf("shouldHonorClusterTLSProfile(%v) = %v, want %v", tt.adherence, result, tt.shouldHonor)
			}
		})
	}
}

func TestTLSProfileToFlags_SkipsUnknownCiphers(t *testing.T) {
	profile := &configv1.TLSSecurityProfile{
		Type: configv1.TLSProfileCustomType,
		Custom: &configv1.CustomTLSProfile{
			TLSProfileSpec: configv1.TLSProfileSpec{
				MinTLSVersion: configv1.VersionTLS12,
				Ciphers: []string{
					"TLS_AES_128_GCM_SHA256",      // Valid
					"UNKNOWN_CIPHER_SUITE",        // Invalid - should be skipped
					"ECDHE-RSA-AES128-GCM-SHA256", // Valid
					"ANOTHER_UNKNOWN_CIPHER",      // Invalid - should be skipped
				},
			},
		},
	}

	_, cipherSuites := TLSProfileToFlags(profile)

	// Should only have 2 ciphers (the valid ones)
	cipherList := strings.Split(cipherSuites, ",")
	if len(cipherList) != 2 {
		t.Errorf("TLSProfileToFlags() with unknown ciphers returned %d ciphers, want 2", len(cipherList))
	}

	// Verify the known ciphers are present
	expectedCiphers := []string{
		"TLS_AES_128_GCM_SHA256",
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	}
	for _, expected := range expectedCiphers {
		found := false
		for _, cipher := range cipherList {
			if cipher == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected cipher %s not found in result: %s", expected, cipherSuites)
		}
	}
}

func TestFilterOutTLSArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name: "Remove all TLS args",
			args: []string{
				"--health-probe-bind-address=:8081",
				"--tls-min-version=VersionTLS12",
				"--tls-cipher-suites=TLS_AES_128_GCM_SHA256",
				"--metrics-bind-address=:8443",
			},
			expected: []string{
				"--health-probe-bind-address=:8081",
				"--metrics-bind-address=:8443",
			},
		},
		{
			name: "Handle mixed TLS args",
			args: []string{
				"--tls-min-version=VersionTLS13",
				"--other-flag=value",
				"--tls-cipher-suites=ciphers",
			},
			expected: []string{
				"--other-flag=value",
			},
		},
		{
			name: "No TLS args to remove",
			args: []string{
				"--health-probe-bind-address=:8081",
				"--metrics-bind-address=:8443",
			},
			expected: []string{
				"--health-probe-bind-address=:8081",
				"--metrics-bind-address=:8443",
			},
		},
		{
			name:     "Empty args",
			args:     []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterOutTLSArgs(tt.args)
			if len(result) != len(tt.expected) {
				t.Errorf("filterOutTLSArgs() returned %d args, want %d", len(result), len(tt.expected))
			}
			for i, arg := range result {
				if i >= len(tt.expected) || arg != tt.expected[i] {
					t.Errorf("filterOutTLSArgs() result[%d] = %v, want %v", i, arg, tt.expected[i])
				}
			}
		})
	}
}
