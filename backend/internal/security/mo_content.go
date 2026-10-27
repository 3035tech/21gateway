package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"strconv"
)

// Separate key and domain binding prevent interchange with account tokens.
type MOContentVault struct {
	vault     *TokenVault
	digestKey []byte
}

func NewMOContentVault(key string) (*MOContentVault, error) {
	return NewMOContentVaultWithKeys(key, "", nil, "")
}
func NewMOContentVaultWithKeys(legacy, active string, keys map[string]string, fingerprintKey string) (*MOContentVault, error) {
	v, err := NewTokenVaultWithKeys(legacy, active, keys)
	if err != nil {
		return nil, err
	}
	if fingerprintKey == "" {
		fingerprintKey = legacy
	}
	raw, err := base64.StdEncoding.DecodeString(fingerprintKey)
	if err != nil || (v.Enabled() && len(raw) != providerKeyBytes) {
		return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	return &MOContentVault{vault: v, digestKey: raw}, nil
}
func (v *MOContentVault) Current(encrypted string) bool {
	return v != nil && v.vault.Current(encrypted)
}

func (v *MOContentVault) Enabled() bool { return v != nil && v.vault.Enabled() }
func moBinding(credentialID uint64, eventKey string) string {
	return "mo:" + strconv.FormatUint(credentialID, 10) + ":" + eventKey
}
func (v *MOContentVault) Encrypt(credentialID uint64, eventKey, message string) (string, error) {
	if !v.Enabled() {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	return v.vault.keys.encrypt(moBinding(credentialID, eventKey), message)
}
func (v *MOContentVault) Decrypt(credentialID uint64, eventKey, encrypted string) (string, error) {
	if v == nil || v.vault == nil || v.vault.keys == nil {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	return v.vault.keys.decrypt(moBinding(credentialID, eventKey), encrypted)
}

func (v *MOContentVault) Fingerprint(credentialID uint64, eventKey string, data []byte) (string, error) {
	if !v.Enabled() {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	mac := hmac.New(sha256.New, v.digestKey)
	mac.Write([]byte(moBinding(credentialID, eventKey)))
	mac.Write([]byte{0})
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil)), nil
}
