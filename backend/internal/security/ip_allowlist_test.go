package security

import (
	"testing"
)

func TestIPAllowlistSupportsExactIPsAndNetworks(t *testing.T) {
	values, err := NormalizeAllowedIPs([]string{"203.0.113.5", "198.51.100.9/24", "2001:db8::/32", "203.0.113.5"})
	if err != nil || len(values) != 3 || values[1] != "198.51.100.0/24" {
		t.Fatal("normalization failed")
	}
	for _, address := range []string{"203.0.113.5:1234", "198.51.100.42:1234", "[2001:db8::1]:1234", "[::ffff:203.0.113.5]:1234"} {
		if !IPAllowed(address, values) {
			t.Error("allowed peer rejected", address)
		}
	}
	for _, address := range []string{"203.0.113.6:1234", "198.51.101.1:1234", "hostname:1234", "invalid"} {
		if IPAllowed(address, values) {
			t.Error("unknown peer allowed", address)
		}
	}
	if _, err := NormalizeAllowedIPs([]string{"host.example"}); err == nil {
		t.Fatal("hostname allowed in IP whitelist")
	}
}
