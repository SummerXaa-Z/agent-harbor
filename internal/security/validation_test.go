package security

import "testing"

func TestValidateOutboundEndpointRejectsPrivateHostsByDefault(t *testing.T) {
	err := ValidateOutboundEndpoint("http://127.0.0.1:8080/mcp")
	if err == nil {
		t.Fatalf("expected loopback endpoint to be rejected by default")
	}
}

func TestValidateOutboundEndpointAllowsPrivateHostsWhenExplicit(t *testing.T) {
	for _, endpoint := range []string{
		"http://127.0.0.1:8080/mcp",
		"http://localhost:8080/mcp",
		"http://10.0.0.2:8080/mcp",
	} {
		t.Run(endpoint, func(t *testing.T) {
			err := ValidateOutboundEndpoint(endpoint, EndpointValidationOptions{AllowPrivateHosts: true})
			if err != nil {
				t.Fatalf("expected private endpoint to be allowed with explicit option, got %v", err)
			}
		})
	}
}
