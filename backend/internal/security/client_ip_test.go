package security

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPResolverTrustsOnlyConfiguredPeer(t *testing.T) {
	resolver, err := NewClientIPResolver([]string{"10.42.0.0/24", "2001:db8::/64"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, peer, header, want string
	}{
		{"trusted IPv4", "10.42.0.3:1234", "198.51.100.4", "198.51.100.4"},
		{"trusted IPv6", "[2001:db8::3]:1234", "2001:db8:1::4", "2001:db8:1::4"},
		{"untrusted peer", "203.0.113.5:1234", "198.51.100.4", "203.0.113.5"},
		{"missing header", "10.42.0.3:1234", "", "10.42.0.3"},
		{"list rejected", "10.42.0.3:1234", "198.51.100.4, 203.0.113.5", "10.42.0.3"},
		{"invalid header", "10.42.0.3:1234", "invalid", "10.42.0.3"},
		{"unspecified header", "10.42.0.3:1234", "0.0.0.0", "10.42.0.3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/", nil)
			request.RemoteAddr = tc.peer
			if tc.header != "" {
				request.Header.Set("CF-Connecting-IP", tc.header)
			}
			request.Header.Set("X-Forwarded-For", "192.0.2.99")
			if got := resolver.Resolve(request); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClientIPResolverRejectsInvalidTrustConfiguration(t *testing.T) {
	for _, value := range []string{"bad", "0.0.0.0/0", "::/0", "::ffff:192.0.2.1/128"} {
		if _, err := NewClientIPResolver([]string{value}); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}
