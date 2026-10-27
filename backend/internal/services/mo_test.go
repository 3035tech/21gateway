package services

import (
	"context"
	"encoding/base64"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"strings"
	"testing"
	"time"
)

type moMemory struct{ events []models.MOEvent }

func (r *moMemory) Accept(_ context.Context, e models.MOEvent) error {
	r.events = append(r.events, e)
	return nil
}
func testMOVault(t *testing.T) *security.MOContentVault {
	t.Helper()
	v, err := security.NewMOContentVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("m", 32))))
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func moInput() dtos.MOInput {
	in := dtos.MOInput{}
	in.Body.EventKey = "mo-1"
	in.Body.CredentialID = 7
	in.Body.ProviderMessageID = "550e8400-e29b-41d4-a716-446655440000"
	in.Body.PhoneNumber = "5511998877665"
	in.Body.Message = "resposta\nOlá 😊"
	in.Body.ReceivedAt = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	return in
}
func TestMOEncryptsAndFingerprintsNormalizedInput(t *testing.T) {
	repo := &moMemory{}
	vault := testMOVault(t)
	s := NewMOService(repo, vault)
	in := moInput()
	for range 2 {
		if err := s.Accept(context.Background(), in); err != nil {
			t.Fatal(err)
		}
	}
	a, b := repo.events[0], repo.events[1]
	if a.EncryptedMessage == b.EncryptedMessage || a.Fingerprint != b.Fingerprint || strings.Contains(a.EncryptedMessage, in.Body.Message) {
		t.Fatal("unsafe encryption or unstable fingerprint")
	}
	plain, err := vault.Decrypt(a.CredentialID, a.EventKey, a.EncryptedMessage)
	if err != nil || plain != in.Body.Message {
		t.Fatal("content not preserved", err)
	}
	if _, err := vault.Decrypt(8, a.EventKey, a.EncryptedMessage); err == nil {
		t.Fatal("cross-account decryption")
	}
	if _, err := vault.Decrypt(7, "other", a.EncryptedMessage); err == nil {
		t.Fatal("cross-event decryption")
	}
	in.Body.PhoneNumber = "5511999999999"
	if err := s.Accept(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if repo.events[2].Fingerprint == a.Fingerprint {
		t.Fatal("changed payload fingerprint unchanged")
	}
}
func TestMOUnavailableKeyAndInvalidPayloadNeverPersist(t *testing.T) {
	repo := &moMemory{}
	empty, _ := security.NewMOContentVault("")
	if err := NewMOService(repo, empty).Accept(context.Background(), moInput()); err == nil {
		t.Fatal("missing key accepted")
	}
	s := NewMOService(repo, testMOVault(t))
	for _, change := range []func(*dtos.MOInput){func(i *dtos.MOInput) { i.Body.CredentialID = 0 }, func(i *dtos.MOInput) { i.Body.EventKey = " " }, func(i *dtos.MOInput) { i.Body.Message = string([]byte{0xff}) }, func(i *dtos.MOInput) { i.Body.ProviderMessageID = "invalid" }, func(i *dtos.MOInput) { i.Body.PhoneNumber = "+5511998877665" }, func(i *dtos.MOInput) { i.Body.Message = strings.Repeat("x", MaximumMOMessageBytes+1) }, func(i *dtos.MOInput) { i.Body.ReceivedAt = time.Now().Add(time.Hour) }} {
		input := moInput()
		change(&input)
		if err := s.Accept(context.Background(), input); err == nil {
			t.Fatal("invalid payload accepted")
		}
	}
	if len(repo.events) != 0 {
		t.Fatal("invalid input persisted")
	}
}
