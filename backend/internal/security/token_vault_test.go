package security

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestTokenVaultBindsAccountAndAuthenticatesCiphertext(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	vault, err := NewTokenVault(key)
	if err != nil {
		t.Fatal(err)
	}
	a, err := vault.Encrypt("account-a", "private-provider-token")
	if err != nil {
		t.Fatal(err)
	}
	b, err := vault.Encrypt("account-a", "private-provider-token")
	if err != nil || a == b || strings.Contains(a, "private-provider-token") {
		t.Fatal("encryption must use random authenticated nonces")
	}
	token, err := vault.Decrypt("account-a", a)
	if err != nil || token != "private-provider-token" {
		t.Fatal("token round trip failed")
	}
	if _, err := vault.Decrypt("account-b", a); err == nil {
		t.Fatal("ciphertext moved to another account")
	}
	if _, err := vault.Decrypt("account-a", a[:len(a)-1]+"!"); err == nil {
		t.Fatal("corrupt ciphertext accepted")
	}
	if _, err := vault.Encrypt("account-a", "token\ninjected"); err == nil {
		t.Fatal("header injection accepted")
	}
}
func TestTokenVaultMissingKeyFailsClosed(t *testing.T) {
	vault, err := NewTokenVault("")
	if err != nil || vault.Enabled() {
		t.Fatal("missing key unexpectedly enabled encryption")
	}
	if _, err := vault.Encrypt("account", "token"); err == nil {
		t.Fatal("plaintext storage permitted")
	}
	if _, err := NewTokenVault("not-base64"); err == nil {
		t.Fatal("malformed key permitted")
	}
}
