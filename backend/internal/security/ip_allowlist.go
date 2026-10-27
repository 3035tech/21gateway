package security

import (
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"net"
	"net/netip"
	"strings"
)

const MaximumAllowedIPs = 100

func NormalizeAllowedIPs(values []string) ([]string, error) {
	if len(values) > MaximumAllowedIPs {
		return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			address, e := netip.ParseAddr(value)
			if e != nil || address.Zone() != "" {
				return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
			}
			address = address.Unmap()
			prefix = netip.PrefixFrom(address, address.BitLen())
		}
		if prefix.Addr().Zone() != "" || prefix.Addr().Is4In6() {
			return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
		}
		value = prefix.Masked().String()
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result, nil
}
func IPAllowed(remoteAddr string, allowed []string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	address = address.Unmap()
	for _, entry := range allowed {
		prefix, err := netip.ParsePrefix(entry)
		if err == nil && prefix.Contains(address) {
			return true
		}
	}
	return false
}
