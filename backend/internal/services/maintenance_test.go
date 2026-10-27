package services

import (
	"context"
	"encoding/base64"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"strings"
	"testing"
	"time"
)

type maintenanceMemory struct {
	company    models.CompanyCiphertext
	credential models.CredentialCiphertext
	mo         models.MOCiphertext
	writes     int
	conflict   bool
	cutoff     time.Time
	applied    bool
	purges     int
}

func (r *maintenanceMemory) CompanyCiphertexts(_ context.Context, after uint64, _ int) ([]models.CompanyCiphertext, error) {
	if r.company.ID == 0 || after >= r.company.ID {
		return nil, nil
	}
	return []models.CompanyCiphertext{r.company}, nil
}
func (r *maintenanceMemory) ReplaceCompanyCiphertext(_ context.Context, _ uint64, old, next string) (bool, error) {
	if r.conflict || r.company.ProviderTokenEncrypted != old {
		return false, nil
	}
	r.company.ProviderTokenEncrypted = next
	r.writes++
	return true, nil
}

func (r *maintenanceMemory) CredentialCiphertexts(_ context.Context, after uint64, _ int) ([]models.CredentialCiphertext, error) {
	if after >= r.credential.ID {
		return nil, nil
	}
	return []models.CredentialCiphertext{r.credential}, nil
}
func (r *maintenanceMemory) MOCiphertexts(_ context.Context, after uint64, _ int) ([]models.MOCiphertext, error) {
	if after >= r.mo.ID {
		return nil, nil
	}
	return []models.MOCiphertext{r.mo}, nil
}
func (r *maintenanceMemory) ReplaceCredentialCiphertext(_ context.Context, _ uint64, old, next string) (bool, error) {
	if r.conflict || r.credential.ProviderTokenEncrypted != old {
		return false, nil
	}
	r.credential.ProviderTokenEncrypted = next
	r.writes++
	return true, nil
}
func (r *maintenanceMemory) ReplaceMOCiphertext(_ context.Context, _ uint64, old, next string) (bool, error) {
	if r.conflict || r.mo.EncryptedMessage != old {
		return false, nil
	}
	r.mo.EncryptedMessage = next
	r.writes++
	return true, nil
}
func (r *maintenanceMemory) PurgeMO(_ context.Context, before time.Time, _ int, apply bool) (int64, error) {
	r.cutoff = before
	r.applied = apply
	r.purges++
	return 1, nil
}
func TestMaintenanceRotationDryRunAndResumption(t *testing.T) {
	old := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("o", 32)))
	next := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("n", 32)))
	oldTokens, _ := security.NewTokenVault(old)
	oldMO, _ := security.NewMOContentVault(old)
	token, _ := oldTokens.Encrypt("client", "token")
	content, _ := oldMO.Encrypt(7, "event", "texto\n😊")
	tokens, _ := security.NewTokenVaultWithKeys(old, "next", map[string]string{"next": next})
	mo, _ := security.NewMOContentVaultWithKeys(old, "next", map[string]string{"next": next}, "")
	repo := &maintenanceMemory{credential: models.CredentialCiphertext{ID: 1, Username: "client", ProviderTokenEncrypted: token}, mo: models.MOCiphertext{ID: 1, CredentialID: 7, EventKey: "event", EncryptedMessage: content}}
	service := NewMaintenanceService(repo, tokens, mo)
	result, err := service.Rotate(context.Background(), false)
	if err != nil || result.Eligible != 2 || result.Changed != 0 || repo.writes != 0 {
		t.Fatal("dry-run wrote data", err, result)
	}
	result, err = service.Rotate(context.Background(), true)
	if err != nil || result.Changed != 2 {
		t.Fatal("rotation failed", err, result)
	}
	if plain, err := tokens.Decrypt("client", repo.credential.ProviderTokenEncrypted); err != nil || plain != "token" {
		t.Fatal("token lost")
	}
	if plain, err := mo.Decrypt(7, "event", repo.mo.EncryptedMessage); err != nil || plain != "texto\n😊" {
		t.Fatal("content lost")
	}
	result, err = service.Rotate(context.Background(), true)
	if err != nil || result.Eligible != 0 || result.Changed != 0 {
		t.Fatal("rotation not resumable", err, result)
	}
	repo.credential.ProviderTokenEncrypted = token
	repo.conflict = true
	result, err = service.Rotate(context.Background(), true)
	if err != nil || result.Conflicts != 1 || repo.credential.ProviderTokenEncrypted != token {
		t.Fatal("concurrent edit overwritten", err, result)
	}
}

func TestMaintenanceRotatesCompanyToken(t *testing.T) {
	old := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("o", 32)))
	next := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("n", 32)))
	previous, _ := security.NewTokenVault(old)
	encrypted, _ := previous.Encrypt("company:42", "provider-secret")
	vault, _ := security.NewTokenVaultWithKeys(old, "next", map[string]string{"next": next})
	repo := &maintenanceMemory{company: models.CompanyCiphertext{ID: 42, ProviderTokenEncrypted: encrypted}}
	result, err := NewMaintenanceService(repo, vault, nil).Rotate(context.Background(), true)
	if err != nil || result.Changed != 1 {
		t.Fatal(result, err)
	}
	if plain, err := vault.Decrypt("company:42", repo.company.ProviderTokenEncrypted); err != nil || plain != "provider-secret" {
		t.Fatal("company token rotation failed", err)
	}
}
func TestRetentionUsesConfirmed90DaysAndDryRun(t *testing.T) {
	repo := &maintenanceMemory{}
	service := NewMaintenanceService(repo, nil, nil)
	before := time.Now().UTC().Add(-90 * 24 * time.Hour)
	count, err := service.PurgeMO(context.Background(), 90, false)
	if err != nil || count != 1 || repo.applied || repo.cutoff.Before(before) || repo.cutoff.After(before.Add(time.Second)) {
		t.Fatal("wrong retention", err)
	}
	_, err = service.PurgeMO(context.Background(), 0, true)
	if err != nil || repo.purges != 1 {
		t.Fatal("disabled retention ran")
	}
	if _, err := service.PurgeMO(context.Background(), -1, true); err == nil {
		t.Fatal("negative retention accepted")
	}
}
