package security

import (
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"strconv"
	"unicode/utf8"
)

const maximumScheduledContentBytes = 4096

type ScheduledContentVault struct{ keys *encryptionKeys }

func NewScheduledContentVault(legacy, active string, keys map[string]string) (*ScheduledContentVault, error) {
	ring, err := newEncryptionKeys(legacy, active, keys)
	if err != nil {
		return nil, err
	}
	return &ScheduledContentVault{keys: ring}, nil
}
func (v *ScheduledContentVault) Enabled() bool { return v != nil && v.keys.enabled() }
func scheduleBinding(credentialID uint64, key string) string {
	return "scheduled-sms:" + strconv.FormatUint(credentialID, 10) + ":" + key
}
func (v *ScheduledContentVault) Encrypt(credentialID uint64, key, text string) (string, error) {
	if !v.Enabled() {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	if credentialID == 0 || key == "" || !utf8.ValidString(text) || len(text) == 0 || len(text) > maximumScheduledContentBytes {
		return "", apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	return v.keys.encrypt(scheduleBinding(credentialID, key), text)
}
func (v *ScheduledContentVault) Decrypt(credentialID uint64, key, text string) (string, error) {
	if !v.Enabled() {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	return v.keys.decrypt(scheduleBinding(credentialID, key), text)
}
