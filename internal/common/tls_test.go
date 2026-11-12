package common

import (
	"strings"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
)

func TestConvertOpenSSLCurveToIANA(t *testing.T) {
	tests := []struct {
		name          string
		openSSLCurve  string
		expectedIANA  string
		expectedFound bool
	}{
		{"X25519", "X25519", "X25519", true},
		{"prime256v1", "prime256v1", "secp256r1", true},
		{"secp384r1", "secp384r1", "secp384r1", true},
		{"secp521r1", "secp521r1", "secp521r1", true},
		{"X448 not supported", "X448", "", false},
		{"P-256", "P-256", "secp256r1", true},
		{"Unknown curve", "unknown-curve", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ianaCurve, found := ConvertOpenSSLCurveToIANA(tt.openSSLCurve)
			if found != tt.expectedFound {
				t.Errorf("ConvertOpenSSLCurveToIANA(%q) found = %v, want %v", tt.openSSLCurve, found, tt.expectedFound)
			}
			if ianaCurve != tt.expectedIANA {
				t.Errorf("ConvertOpenSSLCurveToIANA(%q) = %q, want %q", tt.openSSLCurve, ianaCurve, tt.expectedIANA)
			}
		})
	}
}

func TestTLSProfileToFlags(t *testing.T) {
	tests := []struct {
		name               string
		profile            *configv1.TLSSecurityProfile
		expectedMinVersion string
		expectedCiphers    string
		expectedCurves     string
	}{
		{
			name:               "nil profile",
			profile:            nil,
			expectedMinVersion: "",
			expectedCiphers:    "",
			expectedCurves:     "",
		},
		{
			name: "Intermediate profile",
			profile: &configv1.TLSSecurityProfile{
				Type:         configv1.TLSProfileIntermediateType,
				Intermediate: &configv1.IntermediateTLSProfile{},
			},
			expectedMinVersion: "VersionTLS12",
			// OpenSSL format converted to IANA format
			expectedCiphers: "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
			expectedCurves:  "X25519MLKEM768,X25519,secp256r1,secp384r1",
		},
		{
			name: "Modern profile",
			profile: &configv1.TLSSecurityProfile{
				Type:   configv1.TLSProfileModernType,
				Modern: &configv1.ModernTLSProfile{},
			},
			expectedMinVersion: "VersionTLS13",
			expectedCiphers:    "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256",
			expectedCurves:     "X25519MLKEM768,X25519,secp256r1,secp384r1",
		},
		{
			name: "Old profile",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileOldType,
				Old:  &configv1.OldTLSProfile{},
			},
			expectedMinVersion: "VersionTLS10",
			// OpenSSL format converted to IANA format
			expectedCiphers: "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384,TLS_CHACHA20_POLY1305_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384,TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384,TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,TLS_RSA_WITH_AES_128_GCM_SHA256,TLS_RSA_WITH_AES_256_GCM_SHA384,TLS_RSA_WITH_AES_128_CBC_SHA256,TLS_RSA_WITH_AES_256_CBC_SHA256,TLS_RSA_WITH_AES_128_CBC_SHA,TLS_RSA_WITH_AES_256_CBC_SHA,TLS_RSA_WITH_3DES_EDE_CBC_SHA",
			expectedCurves:  "X25519MLKEM768,X25519,secp256r1,secp384r1",
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
			expectedCurves:     "",
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
			expectedCurves:     "",
		},
		{
			name: "Custom profile with no ciphers",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						MinTLSVersion: configv1.VersionTLS12,
					},
				},
			},
			expectedMinVersion: "VersionTLS12",
			expectedCiphers:    "",
			expectedCurves:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minVersion, cipherSuites, curves := TLSProfileToFlags(tt.profile)
			if minVersion != tt.expectedMinVersion {
				t.Errorf("expected minVersion %q, got %q", tt.expectedMinVersion, minVersion)
			}
			if cipherSuites != tt.expectedCiphers {
				t.Errorf("expected cipherSuites %q, got %q", tt.expectedCiphers, cipherSuites)
			}
			if curves != tt.expectedCurves {
				t.Errorf("expected curves %q, got %q", tt.expectedCurves, curves)
			}
		})
	}
}

func TestValidateTLSProfile(t *testing.T) {
	tests := []struct {
		name      string
		profile   *configv1.TLSSecurityProfile
		expectErr bool
	}{
		{
			name:      "Nil profile",
			profile:   nil,
			expectErr: false,
		},
		{
			name:      "Modern profile",
			profile:   &configv1.TLSSecurityProfile{Type: configv1.TLSProfileModernType},
			expectErr: false,
		},
		{
			name:      "Intermediate profile",
			profile:   &configv1.TLSSecurityProfile{Type: configv1.TLSProfileIntermediateType},
			expectErr: false,
		},
		{
			name:      "Old profile",
			profile:   &configv1.TLSSecurityProfile{Type: configv1.TLSProfileOldType},
			expectErr: false,
		},
		{
			name: "Valid custom profile",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						MinTLSVersion: configv1.VersionTLS12,
						Ciphers:       []string{"TLS_AES_128_GCM_SHA256"},
					},
				},
			},
			expectErr: false,
		},
		{
			name: "Custom profile with no Custom field",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
			},
			expectErr: true,
		},
		{
			name: "Custom profile with no min version",
			profile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{},
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSProfile(tt.profile)
			if (err != nil) != tt.expectErr {
				t.Errorf("ValidateTLSProfile() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestTLSProfileToFlags_WithGroups(t *testing.T) {
	tests := []struct {
		name           string
		groups         []configv1.TLSGroup
		expectedCurves string
	}{
		{
			name:           "Single curve X25519",
			groups:         []configv1.TLSGroup{"X25519"},
			expectedCurves: "X25519",
		},
		{
			name:           "Multiple curves",
			groups:         []configv1.TLSGroup{"X25519", "prime256v1", "secp384r1"},
			expectedCurves: "X25519,secp256r1,secp384r1",
		},
		{
			name:           "All supported curves",
			groups:         []configv1.TLSGroup{"X25519", "X448", "prime256v1", "secp384r1", "secp521r1"},
			expectedCurves: "X25519,secp256r1,secp384r1,secp521r1",
		},
		{
			name:           "Curves with unsupported mixed in",
			groups:         []configv1.TLSGroup{"X25519", "unsupported-curve", "prime256v1"},
			expectedCurves: "X25519,secp256r1",
		},
		{
			name:           "Empty groups",
			groups:         []configv1.TLSGroup{},
			expectedCurves: "",
		},
		{
			name:           "All unsupported curves",
			groups:         []configv1.TLSGroup{"unsupported1", "unsupported2"},
			expectedCurves: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						Ciphers:       []string{"TLS_AES_128_GCM_SHA256"},
						MinTLSVersion: configv1.VersionTLS13,
						Groups:        tt.groups,
					},
				},
			}

			_, _, curves := TLSProfileToFlags(profile)

			if curves != tt.expectedCurves {
				t.Errorf("expected curves %q, got %q", tt.expectedCurves, curves)
			}
		})
	}
}

func TestTLSProfileToFlags_TooManyGroups(t *testing.T) {
	// Test with maximum supported curves
	profile := &configv1.TLSSecurityProfile{
		Type: configv1.TLSProfileCustomType,
		Custom: &configv1.CustomTLSProfile{
			TLSProfileSpec: configv1.TLSProfileSpec{
				Ciphers:       []string{"TLS_AES_128_GCM_SHA256"},
				MinTLSVersion: configv1.VersionTLS13,
				Groups: []configv1.TLSGroup{
					"X25519",
					"prime256v1",
					"secp384r1",
					"secp521r1",
					"X448",
				},
			},
		},
	}

	_, _, curves := TLSProfileToFlags(profile)

	// Should return all 5 supported curves in the order provided
	expected := "X25519,secp256r1,secp384r1,secp521r1"
	if curves != expected {
		t.Errorf("expected curves %q, got %q", expected, curves)
	}
}

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected uint16
	}{
		{"TLS 1.0", "VersionTLS10", 0x0301},
		{"TLS 1.1", "VersionTLS11", 0x0302},
		{"TLS 1.2", "VersionTLS12", 0x0303},
		{"TLS 1.3", "VersionTLS13", 0x0304},
		{"Empty", "", 0},
		{"Unknown", "VersionTLS99", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTLSVersion(tt.version)
			if result != tt.expected {
				t.Errorf("ParseTLSVersion(%q) = 0x%04x, want 0x%04x", tt.version, result, tt.expected)
			}
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name     string
		ciphers  string
		expected int // Number of cipher suites expected
	}{
		{"Empty", "", 0},
		{"Single TLS 1.3", "TLS_AES_128_GCM_SHA256", 1},
		{"Multiple TLS 1.3", "TLS_AES_128_GCM_SHA256,TLS_AES_256_GCM_SHA384", 2},
		{"TLS 1.2 ECDHE", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", 2},
		{"Mixed versions", "TLS_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", 2},
		{"With spaces", "TLS_AES_128_GCM_SHA256 , TLS_AES_256_GCM_SHA384", 2},
		{"Unknown cipher filtered", "TLS_AES_128_GCM_SHA256,UNKNOWN_CIPHER", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCipherSuites(tt.ciphers)
			if len(result) != tt.expected {
				t.Errorf("ParseCipherSuites(%q) returned %d ciphers, want %d", tt.ciphers, len(result), tt.expected)
			}
		})
	}
}

func TestParseCurvePreferences(t *testing.T) {
	tests := []struct {
		name     string
		curves   string
		expected []uint16
	}{
		{"Empty", "", nil},
		{"Single curve", "X25519", []uint16{0x001d}},
		{"Multiple curves", "X25519,secp256r1", []uint16{0x001d, 0x0017}},
		{"All supported", "X25519,secp256r1,secp384r1,secp521r1", []uint16{0x001d, 0x0017, 0x0018, 0x0019}},
		{"With spaces", "X25519 , secp256r1", []uint16{0x001d, 0x0017}},
		{"Unknown filtered", "X25519,unknown_curve,secp256r1", []uint16{0x001d, 0x0017}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCurvePreferences(tt.curves)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseCurvePreferences(%q) returned %d curves, want %d", tt.curves, len(result), len(tt.expected))
				return
			}
			for i, curveID := range result {
				if curveID != tt.expected[i] {
					t.Errorf("ParseCurvePreferences(%q)[%d] = 0x%04x, want 0x%04x", tt.curves, i, curveID, tt.expected[i])
				}
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

	_, cipherSuites, _ := TLSProfileToFlags(profile)

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
			name: "Remove curves arg",
			args: []string{
				"--health-probe",
				"--tls-curve-preferences=X25519,secp256r1",
				"--other-flag",
			},
			expected: []string{
				"--health-probe",
				"--other-flag",
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
				return
			}
			for i, arg := range result {
				if i >= len(tt.expected) || arg != tt.expected[i] {
					t.Errorf("filterOutTLSArgs() result[%d] = %v, want %v", i, arg, tt.expected[i])
				}
			}
		})
	}
}
