package security

import (
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"strings"
)

const providerKeyBytes = 32
const tokenEnvelopeVersion = "v1:"
const maximumProviderTokenBytes = 4096

// Versioned AES-GCM envelopes retain legacy reads during a staged key rotation.
type TokenVault struct{ keys *encryptionKeys }

func NewTokenVault(key string) (*TokenVault, error) { return NewTokenVaultWithKeys(key, "", nil) }
func NewTokenVaultWithKeys(legacy, active string, keys map[string]string) (*TokenVault, error) {
	v, err := newEncryptionKeys(legacy, active, keys)
	if err != nil {
		return nil, err
	}
	return &TokenVault{keys: v}, nil
}
func (v *TokenVault) Enabled() bool                 { return v != nil && v.keys.enabled() }
func (v *TokenVault) Current(encrypted string) bool { return v != nil && v.keys.current(encrypted) }
func (v *TokenVault) Encrypt(username, token string) (string, error) {
	if !v.Enabled() {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	if strings.TrimSpace(token) == "" || len(token) > maximumProviderTokenBytes || strings.ContainsAny(token, "\r\n") {
		return "", apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	return v.keys.encrypt(username, token)
}
func (v *TokenVault) Decrypt(username, encrypted string) (string, error) {
	if v == nil {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	text, err := v.keys.decrypt(username, encrypted)
	if err != nil || text == "" {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	return text, nil
}
