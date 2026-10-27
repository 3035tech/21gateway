package security

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/netip"
)

const APIAuthenticationIPLimit = 600
const MaximumAuthenticationIPLimit = 1000000
const AuthenticationFailureLimit = 10

// Do not trust Forwarded/X-Forwarded-For supplied by the caller.
func AuthenticationPeer(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.Unmap().String()
	}
	return "unknown-peer"
}
func AuthenticationKey(namespace, value string) string {
	hash := sha256.Sum256([]byte(value))
	return namespace + ":" + hex.EncodeToString(hash[:])
}
