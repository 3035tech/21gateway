package security

import (
	"strings"
	"testing"
)

func TestSystemPasswordPolicyPreservesPassphrases(t *testing.T) {
	for _, password := range []string{"short", "ab             ", "123456789012345", "PASSWORDPASSWORD", strings.Repeat("a", 20), strings.Repeat(" ", 20), strings.Repeat("x", MaximumPasswordLength+1), string([]byte{0xff})} {
		if ValidNewSystemPassword(password) {
			t.Fatal("weak/invalid password accepted")
		}
	}
	for _, password := range []string{"uma frase longa de acesso", "a different long passphrase", "Segurança com frase longa"} {
		if !ValidNewSystemPassword(password) {
			t.Fatal("passphrase rejected")
		}
	}
}
func TestAuthenticationPeerCannotBeSplitByPortOrMappedIPv4(t *testing.T) {
	for _, remote := range []string{"192.0.2.4:1234", "192.0.2.4:5678", "[::ffff:192.0.2.4]:1234"} {
		if AuthenticationPeer(remote) != "192.0.2.4" {
			t.Fatal("peer normalization", remote)
		}
	}
	if AuthenticationKey("api-account", "private-user") == AuthenticationKey("admin-account", "private-user") {
		t.Fatal("namespace collision")
	}
	if strings.Contains(AuthenticationKey("api-account", "private-user"), "private-user") {
		t.Fatal("raw username in key")
	}
}
