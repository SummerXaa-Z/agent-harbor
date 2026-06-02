package security

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

type EndpointValidationOptions struct {
	AllowPrivateHosts bool
}

var secretKeyFragments = []string{
	"token",
	"secret",
	"password",
	"api_key",
	"apikey",
	"authorization",
	"authheader",
	"cookie",
	"credential",
}

func ContainsSecretLikeKey(value map[string]any) bool {
	for key, nested := range value {
		if IsSecretLikeKey(key) {
			return true
		}
		if child, ok := nested.(map[string]any); ok && ContainsSecretLikeKey(child) {
			return true
		}
	}
	return false
}

func IsSecretLikeKey(key string) bool {
	lower := strings.ToLower(key)
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(lower)
	for _, fragment := range secretKeyFragments {
		normalizedFragment := strings.NewReplacer("-", "", "_", "", " ", "").Replace(fragment)
		if strings.Contains(lower, fragment) || strings.Contains(normalized, normalizedFragment) {
			return true
		}
	}
	return false
}

func ValidateOutboundEndpoint(raw string, options ...EndpointValidationOptions) error {
	validation := EndpointValidationOptions{}
	if len(options) > 0 {
		validation = options[0]
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("endpoint must be an absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("endpoint scheme must be http or https")
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "metadata.google.internal" {
		return fmt.Errorf("endpoint host is not allowed")
	}
	if host == "localhost" {
		if validation.AllowPrivateHosts {
			return nil
		}
		return fmt.Errorf("endpoint host is not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		addr, ok := netip.AddrFromSlice(ip)
		if ok {
			normalized := addr.Unmap()
			if isUnsafeAddress(normalized) {
				if validation.AllowPrivateHosts && (normalized.IsLoopback() || normalized.IsPrivate()) {
					return nil
				}
				return fmt.Errorf("endpoint host is not allowed")
			}
		}
	}
	return nil
}

func isUnsafeAddress(addr netip.Addr) bool {
	return addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsUnspecified()
}
