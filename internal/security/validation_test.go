package security

import (
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

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

func FuzzValidateOutboundEndpoint(f *testing.F) {
	for _, endpoint := range []string{
		"",
		"https://example.com/mcp",
		"gopher://example.com/resource",
		"http://:8080/mcp",
		"http://localhost.:8080/mcp",
		"http://agent.localhost:8080/mcp",
		"http://metadata.google.internal.:80/",
		"http://127.0.0.1:8080/mcp",
		"http://[::ffff:127.0.0.1]:8080/mcp",
		"http://[fe80::1%25en0]:8080/mcp",
	} {
		f.Add(endpoint)
	}

	f.Fuzz(func(t *testing.T, endpoint string) {
		if len(endpoint) > 4096 {
			t.Skip()
		}
		err := ValidateOutboundEndpoint(endpoint)
		trimmed := strings.TrimSpace(endpoint)
		if trimmed == "" || err != nil {
			return
		}

		parsed, parseErr := url.Parse(trimmed)
		if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			t.Fatalf("unsafe endpoint was accepted: %q", endpoint)
		}
		host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
		if host == "" || host == "metadata.google.internal" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
			t.Fatalf("reserved endpoint host was accepted: %q", endpoint)
		}
		if addr, addrErr := netip.ParseAddr(host); addrErr == nil {
			addr = addr.Unmap()
			if addr.Zone() != "" || addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsUnspecified() {
				t.Fatalf("local or private endpoint address was accepted: %q", endpoint)
			}
		}
	})
}
