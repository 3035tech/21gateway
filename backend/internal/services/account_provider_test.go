package services

import (
	"context"
	"encoding/base64"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	smsprovider "github.com/21mobile/sms-gateway/internal/providers/sms"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/jarcoal/httpmock"
	"net/http"
	"strings"
	"testing"
)

type alternateSMSAdapter struct{ calls int }

func (*alternateSMSAdapter) Code() models.Provider { return models.Provider("test-supplier") }
func (a *alternateSMSAdapter) PrepareCredential(_ context.Context, _ models.APICredential) (smsprovider.Sender, error) {
	a.calls++
	return &stubProvider{enabled: true}, nil
}
func (a *alternateSMSAdapter) PrepareCompany(_ context.Context, _ models.Company) (smsprovider.Sender, error) {
	a.calls++
	return &stubProvider{enabled: true}, nil
}

func TestAccountProviderSelectsRegisteredAdapter(t *testing.T) {
	vault, _ := security.NewTokenVault(base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))))
	client, _ := whitelabel.NewWithTransport("https://provider.example", "unused", httpmock.NewMockTransport())
	alternate := &alternateSMSAdapter{}
	registry, err := NewAccountSMSProvider(whitelabel.NewAdapter(client, vault), alternate)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithCredential(context.Background(), models.APICredential{CompanySMSProvider: alternate.Code()})
	if _, err := registry.Prepare(ctx); err != nil || alternate.calls != 1 {
		t.Fatal("route did not select alternate adapter", err)
	}
	if _, err := registry.Prepare(WithCredential(context.Background(), models.APICredential{CompanySMSProvider: "missing"})); err == nil {
		t.Fatal("unknown provider reached an adapter")
	}
}

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
	provider, err := NewAccountSMSProvider(whitelabel.NewAdapter(client, vault))
	if err != nil {
		t.Fatal(err)
	}
	for _, username := range []string{"account-a", "account-b"} {
		expected = username + "-token"
		encrypted, _ := vault.Encrypt(username, expected)
		ctx := WithCredential(context.Background(), models.APICredential{Username: username, ProviderTokenEncrypted: encrypted})
		ready, err := provider.Prepare(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ready.Send(ctx, smsprovider.SendRequest{Type: models.ChannelSMS}); err != nil {
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
	unknown, _ := vault.Encrypt("unknown-provider", "unrelated-token")
	_, err = provider.Prepare(WithCredential(context.Background(), models.APICredential{Username: "unknown-provider", ProviderTokenEncrypted: unknown, CompanySMSProvider: "unregistered"}))
	if err == nil || transport.GetTotalCallCount() != before {
		t.Fatal("unknown provider must fail closed without calling Zoug")
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
	provider, err := NewAccountSMSProvider(whitelabel.NewAdapter(client, vault))
	if err != nil {
		t.Fatal(err)
	}
	companyProvider, err := provider.PrepareCompany(context.Background(), models.Company{ID: companyID, ProviderTokenEncrypted: companyToken})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := companyProvider.Send(context.Background(), smsprovider.SendRequest{Type: models.ChannelSMS}); err != nil {
		t.Fatal("company token test failed", err)
	}
	for _, username := range []string{"api-one", "api-two"} {
		legacy, _ := vault.Encrypt(username, "old-token")
		credential := models.APICredential{Username: username, CompanyID: &companyID, ProviderTokenEncrypted: legacy, CompanyProviderTokenEncrypted: companyToken}
		ctx := WithCredential(context.Background(), credential)
		ready, err := provider.Prepare(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ready.Send(ctx, smsprovider.SendRequest{Type: models.ChannelSMS}); err != nil {
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
