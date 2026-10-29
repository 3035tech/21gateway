package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

func operationEnvironment(t *testing.T) {
	t.Helper()
	for name, value := range map[string]string{"SCHEDULED_SQS_ENABLED": "false", "SCHEDULED_SQS_QUEUE_URL": "", "AWS_REGION": "", "SCHEDULER_ENABLED": "false", "SCHEDULED_CONTENT_KEY": "", "SCHEDULED_ACTIVE_KEY_ID": "", "SCHEDULED_ENCRYPTION_KEYS": "", "API_AUTH_IP_LIMIT_PER_MINUTE": "600", "TRUSTED_PROXY_CIDRS": "", "DATABASE_URL": "postgres://unused/unused", "COOKIE_SECURE": "false", "COOKIE_CROSS_SITE": "false", "CORS_ALLOWED_ORIGINS": "http://localhost:5173", "WHITELABEL_BASE_URL": "https://provider.example", "WHITELABEL_API_TOKEN": "", "CALLBACK_INGRESS_ENABLED": "false", "MO_INGRESS_ENABLED": "false", "CALLBACK_INGRESS_TOKEN": "", "MO_CONTENT_KEY": "", "PROVIDER_TOKEN_KEY": "", "PROVIDER_ACTIVE_KEY_ID": "", "PROVIDER_ENCRYPTION_KEYS": "", "MO_ACTIVE_KEY_ID": "", "MO_ENCRYPTION_KEYS": "", "MO_FINGERPRINT_KEY": "", "MO_RETENTION_DAYS": "90", "OPERATIONS_ENABLED": "false", "OPERATIONS_TOKEN": ""} {
		t.Setenv(name, value)
	}
	t.Setenv("HEALTH_DEPENDENCIES_TOKEN_SHA256", "")
	t.Setenv("SMS_SQS_ENABLED", "false")
	t.Setenv("SMS_SQS_QUEUE_URL", "")
	t.Setenv("ZOUG_CALLBACK_ENABLED", "false")
	t.Setenv("ZOUG_CALLBACK_TOKEN_SHA256", "")
	t.Setenv("ZOUG_CALLBACK_ALLOWED_IPS", "")
}

func TestZougCallbackRequiresHashAndIPAllowlist(t *testing.T) {
	operationEnvironment(t)
	t.Setenv("ZOUG_CALLBACK_ENABLED", "true")
	if _, err := Load(); err == nil {
		t.Fatal("unprotected webhook accepted")
	}
	digest := sha256.Sum256([]byte(strings.Repeat("t", 32)))
	t.Setenv("ZOUG_CALLBACK_TOKEN_SHA256", hex.EncodeToString(digest[:]))
	if _, err := Load(); err == nil {
		t.Fatal("webhook without source allowlist accepted")
	}
	t.Setenv("ZOUG_CALLBACK_ALLOWED_IPS", "18.232.239.74,3.213.146.37,34.195.124.142")
	if _, err := Load(); err != nil {
		t.Fatal("protected webhook rejected", err)
	}
	t.Setenv("ZOUG_CALLBACK_ALLOWED_IPS", "0.0.0.0")
	if _, err := Load(); err == nil {
		t.Fatal("unspecified source allowed")
	}
}
func TestAllSendQueueRequiresSeparateValidTransport(t *testing.T) {
	operationEnvironment(t)
	t.Setenv("SMS_SQS_ENABLED", "true")
	if _, err := Load(); err == nil {
		t.Fatal("queue enabled without encryption and transport")
	}
	t.Setenv("SCHEDULER_ENABLED", "true")
	t.Setenv("WORKERS_IN_API", "false")
	t.Setenv("SCHEDULED_CONTENT_KEY", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("s", 32))))
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("SMS_SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123456789012/all-sms")
	if _, err := Load(); err != nil {
		t.Fatal("valid all-send queue rejected", err)
	}
	t.Setenv("SCHEDULED_SQS_ENABLED", "true")
	t.Setenv("SCHEDULED_SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123456789012/old-batches")
	if _, err := Load(); err == nil {
		t.Fatal("two queue formats activated together")
	}
}
func TestOperationalConfigurationFailsClosed(t *testing.T) {
	operationEnvironment(t)
	cfg, err := Load()
	if err != nil || cfg.MORetentionDays != 90 || cfg.OperationsEnabled {
		t.Fatal("default operational setup invalid")
	}
	t.Setenv("OPERATIONS_ENABLED", "true")
	if _, err := Load(); err == nil {
		t.Fatal("unprotected diagnostics enabled")
	}
	t.Setenv("OPERATIONS_TOKEN", strings.Repeat("t", 32))
	if _, err := Load(); err != nil {
		t.Fatal("protected diagnostics rejected")
	}
	t.Setenv("MO_RETENTION_DAYS", "-1")
	if _, err := Load(); err == nil {
		t.Fatal("negative retention allowed")
	}
	t.Setenv("MO_RETENTION_DAYS", "90")
	t.Setenv("MO_ENCRYPTION_KEYS", "invalid JSON")
	if _, err := Load(); err == nil {
		t.Fatal("malformed keyring allowed")
	}
}

func TestHealthDependenciesTokenHashConfiguration(t *testing.T) {
	operationEnvironment(t)
	for _, value := range []string{"short", strings.Repeat("x", 64), strings.Repeat("A", 64)} {
		t.Setenv("HEALTH_DEPENDENCIES_TOKEN_SHA256", value)
		if _, err := Load(); err == nil {
			t.Fatal("invalid health token hash accepted")
		}
	}
	t.Setenv("HEALTH_DEPENDENCIES_TOKEN_SHA256", strings.Repeat("a", 64))
	if _, err := Load(); err != nil {
		t.Fatal("valid health token hash rejected", err)
	}
}
func TestVersionedMOIngressRequiresStableFingerprintKey(t *testing.T) {
	operationEnvironment(t)
	key := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	t.Setenv("MO_INGRESS_ENABLED", "true")
	t.Setenv("CALLBACK_INGRESS_TOKEN", strings.Repeat("t", 32))
	t.Setenv("MO_ACTIVE_KEY_ID", "next")
	t.Setenv("MO_ENCRYPTION_KEYS", `{"next":"`+key+`"}`)
	if _, err := Load(); err == nil {
		t.Fatal("missing HMAC key allowed")
	}
	t.Setenv("MO_FINGERPRINT_KEY", key)
	if _, err := Load(); err != nil {
		t.Fatal("valid versioned setup rejected")
	}
}

func TestAuthenticationBudgetConfiguration(t *testing.T) {
	operationEnvironment(t)
	for _, value := range []string{"0", "-1", "1000001", "invalid"} {
		t.Setenv("API_AUTH_IP_LIMIT_PER_MINUTE", value)
		if _, err := Load(); err == nil {
			t.Fatal("unsafe authentication budget accepted", value)
		}
	}
	t.Setenv("API_AUTH_IP_LIMIT_PER_MINUTE", "1200")
	cfg, err := Load()
	if err != nil || cfg.APIAuthenticationIPLimit != 1200 {
		t.Fatal("authentication budget override failed", err)
	}
}

func TestTrustedProxyConfiguration(t *testing.T) {
	operationEnvironment(t)
	for _, value := range []string{"bad", "0.0.0.0/0", "::/0"} {
		t.Setenv("TRUSTED_PROXY_CIDRS", value)
		if _, err := Load(); err == nil {
			t.Fatal("unsafe proxy trust accepted", value)
		}
	}
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.42.0.0/24")
	cfg, err := Load()
	if err != nil || len(cfg.TrustedProxyCIDRs) != 1 {
		t.Fatal("valid proxy trust rejected", err)
	}
}

func TestScheduledConfigurationRequiresPrivateContentKey(t *testing.T) {
	operationEnvironment(t)
	t.Setenv("SCHEDULER_ENABLED", "true")
	if _, err := Load(); err == nil {
		t.Fatal("plaintext scheduler allowed")
	}
	t.Setenv("SCHEDULED_CONTENT_KEY", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("s", 32))))
	if _, err := Load(); err != nil {
		t.Fatal("configured scheduler rejected", err)
	}
	t.Setenv("SCHEDULED_ENCRYPTION_KEYS", "invalid")
	if _, err := Load(); err == nil {
		t.Fatal("invalid schedule keyring allowed")
	}
}

func TestSQSConfigurationIsExplicitAndDoesNotAllowArbitraryEndpoints(t *testing.T) {
	operationEnvironment(t)
	t.Setenv("SCHEDULED_SQS_ENABLED", "true")
	if _, err := Load(); err == nil {
		t.Fatal("SQS enabled without scheduler/queue")
	}
	t.Setenv("SCHEDULER_ENABLED", "true")
	t.Setenv("SCHEDULED_CONTENT_KEY", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("s", 32))))
	t.Setenv("AWS_REGION", "us-east-1")
	for _, queue := range []string{"http://sqs.us-east-1.amazonaws.com/123456789012/scheduled", "https://evil.example/123456789012/scheduled", "https://sqs.us-east-1.amazonaws.com/123456789012/scheduled.fifo", "https://sqs.us-east-1.amazonaws.com/not-account/queue", "https://sqs.us-east-1.amazonaws.com/123456789012/queue?token=secret"} {
		t.Setenv("SCHEDULED_SQS_QUEUE_URL", queue)
		if _, err := Load(); err == nil {
			t.Fatal("unsafe/unsupported SQS endpoint accepted")
		}
	}
	t.Setenv("SCHEDULED_SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123456789012/scheduled")
	if _, err := Load(); err != nil {
		t.Fatal("valid standard queue rejected", err)
	}
}
