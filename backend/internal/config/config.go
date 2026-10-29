package config

import (
	"encoding/hex"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/security"
	"net/http"
	"net/mail"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

var scheduledQueuePath = regexp.MustCompile(`^/[0-9]{12}/[A-Za-z0-9_-]{1,80}$`)
var scheduledQueueRegion = regexp.MustCompile(`^[a-z]{2}(-[a-z]+)+-[0-9]+$`)
var testSMSDestination = regexp.MustCompile(`^[0-9]{11}$`)
var smtpHost = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`)

type Config struct {
	WorkersInAPI                  bool     `env:"WORKERS_IN_API" envDefault:"true"`
	SMSSQSEnabled                 bool     `env:"SMS_SQS_ENABLED" envDefault:"false"`
	SMSSQSQueueURL                string   `env:"SMS_SQS_QUEUE_URL"`
	ScheduledSQSEnabled           bool     `env:"SCHEDULED_SQS_ENABLED" envDefault:"false"`
	ScheduledSQSQueueURL          string   `env:"SCHEDULED_SQS_QUEUE_URL"`
	AWSRegion                     string   `env:"AWS_REGION"`
	SchedulerEnabled              bool     `env:"SCHEDULER_ENABLED" envDefault:"false"`
	ScheduledContentKey           string   `env:"SCHEDULED_CONTENT_KEY"`
	ScheduledActiveKeyID          string   `env:"SCHEDULED_ACTIVE_KEY_ID"`
	ScheduledEncryptionKeys       string   `env:"SCHEDULED_ENCRYPTION_KEYS"`
	SMSHistoryContentKey          string   `env:"SMS_HISTORY_CONTENT_KEY"`
	SMSHistoryActiveKeyID         string   `env:"SMS_HISTORY_ACTIVE_KEY_ID"`
	SMSHistoryEncryptionKeys      string   `env:"SMS_HISTORY_ENCRYPTION_KEYS"`
	APIAuthenticationIPLimit      int      `env:"API_AUTH_IP_LIMIT_PER_MINUTE" envDefault:"600"`
	TrustedProxyCIDRs             []string `env:"TRUSTED_PROXY_CIDRS" envSeparator:","`
	OperationsEnabled             bool     `env:"OPERATIONS_ENABLED" envDefault:"false"`
	OperationsToken               string   `env:"OPERATIONS_TOKEN"`
	HealthDependenciesTokenSHA256 string   `env:"HEALTH_DEPENDENCIES_TOKEN_SHA256"`
	MORetentionDays               int      `env:"MO_RETENTION_DAYS" envDefault:"90"`
	ProviderActiveKeyID           string   `env:"PROVIDER_ACTIVE_KEY_ID"`
	ProviderEncryptionKeys        string   `env:"PROVIDER_ENCRYPTION_KEYS"`
	MOActiveKeyID                 string   `env:"MO_ACTIVE_KEY_ID"`
	MOEncryptionKeys              string   `env:"MO_ENCRYPTION_KEYS"`
	MOFingerprintKey              string   `env:"MO_FINGERPRINT_KEY"`
	LogUnexpectedProviderResponse bool     `env:"WHITELABEL_LOG_UNEXPECTED_RESPONSE_BODY" envDefault:"false"`

	MOContentKey                    string   `env:"MO_CONTENT_KEY"`
	MOIngressEnabled                bool     `env:"MO_INGRESS_ENABLED" envDefault:"false"`
	ProviderTokenKey                string   `env:"PROVIDER_TOKEN_KEY"`
	CookieSecure                    bool     `env:"COOKIE_SECURE" envDefault:"true"`
	CookieCrossSite                 bool     `env:"COOKIE_CROSS_SITE" envDefault:"false"`
	CallbackIngressToken            string   `env:"CALLBACK_INGRESS_TOKEN"`
	CallbackIngressEnabled          bool     `env:"CALLBACK_INGRESS_ENABLED" envDefault:"false"`
	ZougCallbackEnabled             bool     `env:"ZOUG_CALLBACK_ENABLED" envDefault:"false"`
	ZougCallbackTokenSHA256         string   `env:"ZOUG_CALLBACK_TOKEN_SHA256"`
	ZougCallbackAllowedIPs          []string `env:"ZOUG_CALLBACK_ALLOWED_IPS" envSeparator:","`
	Address                         string   `env:"HTTP_ADDRESS" envDefault:":8080"`
	DatabaseURL                     string   `env:"DATABASE_URL,required"`
	AllowedOrigins                  []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"http://localhost:5173" envSeparator:","`
	WhiteLabelBaseURL               string   `env:"WHITELABEL_BASE_URL" envDefault:"https://endpoint.devyx.com.br"`
	WhiteLabelToken                 string   `env:"WHITELABEL_API_TOKEN"`
	TestSMSDestination              string   `env:"TEST_SMS_DESTINATION"`
	ZougDiagnosticDestinationSHA256 string   `env:"ZOUG_DIAGNOSTIC_DESTINATION_SHA256"`
	TestSMSAllowAnyDestination      bool     `env:"TEST_SMS_ALLOW_ANY_DESTINATION" envDefault:"false"`
	MFAEncryptionKey                string   `env:"MFA_ENCRYPTION_KEY"`
	RecoverySMTPHost                string   `env:"RECOVERY_SMTP_HOST"`
	RecoverySMTPPort                string   `env:"RECOVERY_SMTP_PORT"`
	RecoverySMTPUser                string   `env:"RECOVERY_SMTP_USER"`
	RecoverySMTPPassword            string   `env:"RECOVERY_SMTP_PASSWORD"`
	RecoverySMTPFrom                string   `env:"RECOVERY_SMTP_FROM"`
	PublicAppURL                    string   `env:"PUBLIC_APP_URL"`
	DocsEnabled                     bool     `env:"DOCS_ENABLED" envDefault:"false"`
}

func (c Config) MFAVault() (*security.TokenVault, error) {
	return security.NewTokenVault(c.MFAEncryptionKey)
}

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, err)
	}
	if cfg.APIAuthenticationIPLimit < 1 || cfg.APIAuthenticationIPLimit > security.MaximumAuthenticationIPLimit {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	if _, err := security.NewClientIPResolver(cfg.TrustedProxyCIDRs); err != nil {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, err)
	}
	if cfg.TestSMSDestination != "" && !testSMSDestination.MatchString(cfg.TestSMSDestination) {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	if cfg.ZougDiagnosticDestinationSHA256 != "" {
		if len(cfg.ZougDiagnosticDestinationSHA256) != 64 {
			return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
		if _, err := hex.DecodeString(cfg.ZougDiagnosticDestinationSHA256); err != nil {
			return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
	}
	if cfg.RecoverySMTPHost != "" || cfg.RecoverySMTPPort != "" || cfg.RecoverySMTPUser != "" || cfg.RecoverySMTPPassword != "" || cfg.RecoverySMTPFrom != "" || cfg.PublicAppURL != "" {
		app, err := url.Parse(cfg.PublicAppURL)
		port, portErr := strconv.Atoi(cfg.RecoverySMTPPort)
		from, fromErr := mail.ParseAddress(cfg.RecoverySMTPFrom)
		if err != nil || app.Scheme != "https" || app.Host == "" || app.User != nil || app.RawQuery != "" || app.Fragment != "" || !smtpHost.MatchString(cfg.RecoverySMTPHost) || portErr != nil || port < 1 || port > 65535 || cfg.RecoverySMTPUser == "" || cfg.RecoverySMTPPassword == "" || fromErr != nil || from.Address != cfg.RecoverySMTPFrom || strings.ContainsAny(cfg.RecoverySMTPUser+cfg.RecoverySMTPPassword, "\r\n") {
			return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
	}
	if cfg.ScheduledSQSEnabled {
		queue, err := url.Parse(cfg.ScheduledSQSQueueURL)
		if err != nil || !cfg.SchedulerEnabled || !scheduledQueueRegion.MatchString(cfg.AWSRegion) || queue.Scheme != "https" || queue.Host != "sqs."+cfg.AWSRegion+".amazonaws.com" || queue.User != nil || queue.RawQuery != "" || queue.Fragment != "" || !scheduledQueuePath.MatchString(queue.Path) || queue.RawPath != "" {
			return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
	}
	if cfg.SMSSQSEnabled {
		queue, err := url.Parse(cfg.SMSSQSQueueURL)
		if err != nil || !cfg.SchedulerEnabled || cfg.ScheduledSQSEnabled || cfg.WorkersInAPI || !scheduledQueueRegion.MatchString(cfg.AWSRegion) || queue.Scheme != "https" || queue.Host != "sqs."+cfg.AWSRegion+".amazonaws.com" || queue.User != nil || queue.RawQuery != "" || queue.Fragment != "" || !scheduledQueuePath.MatchString(queue.Path) || queue.RawPath != "" {
			return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
	}
	parsed, err := url.Parse(cfg.WhiteLabelBaseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || strings.ContainsAny(cfg.WhiteLabelToken, "\r\n") {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	if cfg.CookieCrossSite && !cfg.CookieSecure {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	for _, origin := range cfg.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || strings.ContainsAny(u.Host, "*\r\n") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (cfg.CookieSecure && u.Scheme != "https") {
			return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
		if err := http.NewCrossOriginProtection().AddTrustedOrigin(origin); err != nil {
			return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
	}
	if (cfg.CallbackIngressEnabled || cfg.MOIngressEnabled) && len(cfg.CallbackIngressToken) < 32 {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	if cfg.ZougCallbackEnabled {
		if !validSHA256Hex(cfg.ZougCallbackTokenSHA256) || len(cfg.ZougCallbackAllowedIPs) == 0 {
			return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
		for _, raw := range cfg.ZougCallbackAllowedIPs {
			ip, err := netip.ParseAddr(strings.TrimSpace(raw))
			if err != nil || ip.IsUnspecified() || ip.IsLoopback() {
				return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
			}
		}
	}
	if cfg.MOIngressEnabled && cfg.MOContentKey == "" && cfg.MOActiveKeyID == "" {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	if cfg.MORetentionDays < 0 || cfg.MORetentionDays > MaximumMORetentionDays || (cfg.OperationsEnabled && len(cfg.OperationsToken) < 32) {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	if cfg.HealthDependenciesTokenSHA256 != "" && !validSHA256Hex(cfg.HealthDependenciesTokenSHA256) {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	scheduledVault, err := cfg.ScheduledVault()
	if err != nil {
		return cfg, err
	}
	if cfg.SchedulerEnabled && !scheduledVault.Enabled() {
		return cfg, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	if _, err := cfg.TokenVault(); err != nil {
		return cfg, err
	}
	if _, err := cfg.MOVault(); err != nil {
		return cfg, err
	}
	if _, err := cfg.SMSHistoryVault(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func validSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			if char < 'a' || char > 'f' {
				return false
			}
		}
	}
	return true
}

func parseEncryptionKeys(raw string) (map[string]string, error) {
	keys := map[string]string{}
	if raw == "" {
		return keys, nil
	}
	if err := json.Unmarshal([]byte(raw), &keys); err != nil || keys == nil {
		return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	return keys, nil
}
func (c Config) TokenVault() (*security.TokenVault, error) {
	keys, err := parseEncryptionKeys(c.ProviderEncryptionKeys)
	if err != nil {
		return nil, err
	}
	return security.NewTokenVaultWithKeys(c.ProviderTokenKey, c.ProviderActiveKeyID, keys)
}
func (c Config) MOVault() (*security.MOContentVault, error) {
	keys, err := parseEncryptionKeys(c.MOEncryptionKeys)
	if err != nil {
		return nil, err
	}
	return security.NewMOContentVaultWithKeys(c.MOContentKey, c.MOActiveKeyID, keys, c.MOFingerprintKey)
}

func (c Config) ScheduledVault() (*security.ScheduledContentVault, error) {
	keys, err := parseEncryptionKeys(c.ScheduledEncryptionKeys)
	if err != nil {
		return nil, err
	}
	return security.NewScheduledContentVault(c.ScheduledContentKey, c.ScheduledActiveKeyID, keys)
}
func (c Config) SMSHistoryVault() (*security.SMSHistoryVault, error) {
	keys, err := parseEncryptionKeys(c.SMSHistoryEncryptionKeys)
	if err != nil {
		return nil, err
	}
	return security.NewSMSHistoryVault(c.SMSHistoryContentKey, c.SMSHistoryActiveKeyID, keys)
}
