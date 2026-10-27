package security

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestEncryptionKeyRotationKeepsLegacyReads(t *testing.T) {
	old := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("o", 32)))
	next := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("n", 32)))
	legacy, _ := NewTokenVault(old)
	first, err := legacy.Encrypt("account", "secret")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewTokenVaultWithKeys(old, "", map[string]string{"next": next})
	if err != nil {
		t.Fatal(err)
	}
	writer, err := NewTokenVaultWithKeys(old, "next", map[string]string{"next": next})
	if err != nil {
		t.Fatal(err)
	}
	if value, err := writer.Decrypt("account", first); err != nil || value != "secret" {
		t.Fatal("legacy data lost")
	}
	second, err := writer.Encrypt("account", "secret")
	if err != nil || !strings.HasPrefix(second, "v2:next:") {
		t.Fatal("missing active key ID")
	}
	if value, err := reader.Decrypt("account", second); err != nil || value != "secret" {
		t.Fatal("preloaded reader cannot read new ciphertext")
	}
	if _, err := legacy.Decrypt("account", second); err == nil {
		t.Fatal("old binary silently accepts new format")
	}
	if _, err := writer.Decrypt("different", second); err == nil {
		t.Fatal("binding not enforced")
	}
	withoutOld, err := NewTokenVaultWithKeys("", "next", map[string]string{"next": next})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := withoutOld.Decrypt("account", first); err == nil {
		t.Fatal("removed old key still available")
	}
	for _, test := range []struct {
		active string
		keys   map[string]string
	}{{"missing", map[string]string{"next": next}}, {"bad:id", map[string]string{"bad:id": next}}, {"next", map[string]string{"next": "invalid"}}} {
		if _, err := NewTokenVaultWithKeys(old, test.active, test.keys); err == nil {
			t.Fatal("invalid keyring allowed")
		}
	}
	moOld, _ := NewMOContentVault(old)
	moNew, err := NewMOContentVaultWithKeys(old, "next", map[string]string{"next": next}, "")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := moOld.Fingerprint(7, "event", []byte("payload"))
	b, _ := moNew.Fingerprint(7, "event", []byte("payload"))
	if a != b {
		t.Fatal("encryption rotation broke deduplication")
	}
	encrypted, _ := moOld.Encrypt(7, "event", "Olá\n😊")
	if text, err := moNew.Decrypt(7, "event", encrypted); err != nil || text != "Olá\n😊" {
		t.Fatal("MO legacy read lost")
	}
	if _, err := NewMOContentVaultWithKeys("", "next", map[string]string{"next": next}, ""); err == nil {
		t.Fatal("missing stable fingerprint key allowed")
	}
}
