package security

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestSMSHistoryBinding(t *testing.T) {
	vault, err := NewSMSHistoryVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("h", 32))), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := vault.Encrypt(7, "message-1", "Ola, codigo 1234")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(encrypted, "1234") {
		t.Fatal("plaintext stored")
	}
	if text, err := vault.Decrypt(7, "message-1", encrypted); err != nil || text != "Ola, codigo 1234" {
		t.Fatal("content changed", err)
	}
	if _, err := vault.Decrypt(8, "message-1", encrypted); err == nil {
		t.Fatal("cross-credential decrypt")
	}
	if _, err := vault.Decrypt(7, "message-2", encrypted); err == nil {
		t.Fatal("cross-message decrypt")
	}
}
