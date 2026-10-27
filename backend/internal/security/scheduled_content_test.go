package security

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestScheduledContentBindingAndMultiline(t *testing.T) {
	vault, err := NewScheduledContentVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("s", 32))), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := vault.Encrypt(7, "job-1", "linha um\nlinha dois")
	if err != nil {
		t.Fatal(err)
	}
	if text, err := vault.Decrypt(7, "job-1", encrypted); err != nil || text != "linha um\nlinha dois" {
		t.Fatal("content changed", err)
	}
	if _, err := vault.Decrypt(8, "job-1", encrypted); err == nil {
		t.Fatal("cross-account decrypt")
	}
	if _, err := vault.Decrypt(7, "job-2", encrypted); err == nil {
		t.Fatal("cross-job decrypt")
	}
}
