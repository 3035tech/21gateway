package security

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strings"
)

type clientIPContextKey struct{}

var errInvalidProxyCIDR = errors.New("invalid trusted proxy CIDR")

// ClientIPResolver accepts Cloudflare's single-IP header only from explicitly
// trusted connection peers. An empty trust list keeps using RemoteAddr.
type ClientIPResolver struct{ trusted []netip.Prefix }

func NewClientIPResolver(cidrs []string) (ClientIPResolver, error) {
	resolver := ClientIPResolver{trusted: make([]netip.Prefix, 0, len(cidrs))}
	for _, raw := range cidrs {
		raw = strings.TrimSpace(raw)
		if raw == "" && len(cidrs) == 1 {
			continue
		}
		prefix, err := netip.ParsePrefix(raw)
		if err != nil || prefix.Addr().Zone() != "" || prefix.Addr().Is4In6() || prefix.Bits() == 0 {
			return ClientIPResolver{}, errInvalidProxyCIDR
		}
		resolver.trusted = append(resolver.trusted, prefix.Masked())
	}
	return resolver, nil
}

func (r ClientIPResolver) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		ip := r.Resolve(request)
		ctx := context.WithValue(request.Context(), clientIPContextKey{}, ip)
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}

func (r ClientIPResolver) Resolve(request *http.Request) string {
	peer := AuthenticationPeer(request.RemoteAddr)
	address, err := netip.ParseAddr(peer)
	if err != nil {
		return peer
	}
	trusted := false
	for _, prefix := range r.trusted {
		if prefix.Contains(address) {
			trusted = true
			break
		}
	}
	if !trusted {
		return peer
	}
	values := request.Header.Values("CF-Connecting-IP")
	if len(values) != 1 || strings.Contains(values[0], ",") {
		return peer
	}
	client, err := netip.ParseAddr(strings.TrimSpace(values[0]))
	if err != nil || client.Zone() != "" || client.IsUnspecified() {
		return peer
	}
	return client.Unmap().String()
}

func ClientIP(request *http.Request) string {
	if ip, ok := request.Context().Value(clientIPContextKey{}).(string); ok && ip != "" {
		return ip
	}
	return AuthenticationPeer(request.RemoteAddr)
}
