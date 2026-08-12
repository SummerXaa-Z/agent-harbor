package domain

import "testing"

func TestCapabilityFingerprintIsCanonicalAndPolicySensitive(t *testing.T) {
	capability := Capability{
		ID:              "cap-ticket-read",
		TargetID:        "agt-support",
		Type:            CapabilityTypeMCPTool,
		Key:             "lookup_ticket",
		Action:          CapabilityActionRead,
		NativeScopes:    []string{"tickets.read", "support.read"},
		DataDomains:     []string{"support", "crm"},
		DataScopes:      []DataScope{{DataDomain: "support", Dataset: "tickets", Field: "title"}, {DataDomain: "support", Dataset: "tickets", Field: "status"}},
		Sensitivity:     CapabilitySensitivityInternal,
		RiskLevel:       CapabilityRiskLow,
		EnforcementMode: CapabilityEnforcementGateway,
		DiscoveryStatus: CapabilityDiscoveryApproved,
		Version:         1,
	}
	reordered := capability
	reordered.NativeScopes = []string{"support.read", "tickets.read"}
	reordered.DataDomains = []string{"crm", "support"}
	reordered.DataScopes = []DataScope{capability.DataScopes[1], capability.DataScopes[0]}
	if CapabilityFingerprint(capability) != CapabilityFingerprint(reordered) {
		t.Fatalf("equivalent capability policy order must not change fingerprint")
	}

	changed := capability
	changed.RiskLevel = CapabilityRiskHigh
	if CapabilityFingerprint(capability) == CapabilityFingerprint(changed) {
		t.Fatalf("risk policy change must change fingerprint")
	}
}
