package services

import (
	"context"
	"encoding/base64"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/jarcoal/httpmock"
	"net/http"
	"strings"
	"testing"
)

func TestPerAccountProviderHasNoSharedTokenFallback(t *testing.T) {
	vault, _ := security.NewTokenVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	transport := httpmock.NewMockTransport()
	expected := ""
	transport.RegisterResponder(http.MethodPost, "https://provider.example/api/send", func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-API-TOKEN") != expected {
			t.Fatal("token crossed account boundary")
		}
		return httpmock.NewStringResponder(200, `{"message_id":"550e8400-e29b-41d4-a716-446655440001","status":"sent"}`)(r)
	})
	client, _ := whitelabel.NewWithTransport("https://provider.example", "must-not-be-used", transport)
	provider := NewAccountSMSProvider(client, vault)
	for _, username := range []string{"account-a", "account-b"} {
		expected = username + "-token"
		encrypted, _ := vault.Encrypt(username, expected)
		ctx := WithCredential(context.Background(), models.APICredential{Username: username, ProviderTokenEncrypted: encrypted})
		ready, err := provider.Prepare(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ready.Send(ctx, whitelabel.SendRequest{}); err != nil {
			t.Fatal("account send failed")
		}
	}
	before := transport.GetTotalCallCount()
	if _, err := provider.Prepare(WithCredential(context.Background(), models.APICredential{Username: "no-token"})); err == nil {
		t.Fatal("shared token fallback enabled")
	}
	if transport.GetTotalCallCount() != before {
		t.Fatal("unconfigured account reached supplier")
	}
}

func TestCompanyTokenSharedAcrossAPICredentials(t *testing.T) {
	vault, _ := security.NewTokenVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	companyID := uint64(42)
	companyToken, _ := vault.Encrypt("company:42", "company-token")
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder(http.MethodPost, "https://provider.example/api/send", func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-API-TOKEN") != "company-token" {
			t.Fatal("wrong provider token")
		}
		return httpmock.NewStringResponder(200, `{"message_id":"550e8400-e29b-41d4-a716-446655440001","status":"sent"}`)(r)
	})
	client, _ := whitelabel.NewWithTransport("https://provider.example", "unused", transport)
	provider := NewAccountSMSProvider(client, vault)
	for _, username := range []string{"api-one", "api-two"} {
		legacy, _ := vault.Encrypt(username, "old-token")
		credential := models.APICredential{Username: username, CompanyID: &companyID, ProviderTokenEncrypted: legacy, CompanyProviderTokenEncrypted: companyToken}
		ctx := WithCredential(context.Background(), credential)
		ready, err := provider.Prepare(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ready.Send(ctx, whitelabel.SendRequest{}); err != nil {
			t.Fatal(err)
		}
		credential.CompanyProviderTokenEncrypted = ""
		if _, err := provider.Prepare(WithCredential(context.Background(), credential)); err == nil {
			t.Fatal("removed company token fell back to legacy token")
		}
		credential.CompanyProviderTokenLegacyFallback = true
		if _, err := provider.Prepare(WithCredential(context.Background(), credential)); err != nil {
			t.Fatal("legacy migration fallback lost", err)
		}
	}
}
