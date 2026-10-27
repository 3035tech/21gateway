package security

import (
	"strconv"
	"unicode/utf8"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

const maximumSMSHistoryBytes = 4096

type SMSHistoryVault struct{ keys *encryptionKeys }

func NewSMSHistoryVault(legacy, active string, keys map[string]string) (*SMSHistoryVault, error) {
	ring, err := newEncryptionKeys(legacy, active, keys)
	if err != nil {
		return nil, err
	}
	return &SMSHistoryVault{keys: ring}, nil
}

func (v *SMSHistoryVault) Enabled() bool { return v != nil && v.keys.enabled() }

func smsHistoryBinding(credentialID uint64, reference string) string {
	return "sms-history:" + strconv.FormatUint(credentialID, 10) + ":" + reference
}

func (v *SMSHistoryVault) Encrypt(credentialID uint64, reference, text string) (string, error) {
	if !v.Enabled() {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	if credentialID == 0 || reference == "" || !utf8.ValidString(text) || len(text) == 0 || len(text) > maximumSMSHistoryBytes {
		return "", apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	return v.keys.encrypt(smsHistoryBinding(credentialID, reference), text)
}

func (v *SMSHistoryVault) Decrypt(credentialID uint64, reference, encrypted string) (string, error) {
	if !v.Enabled() {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	return v.keys.decrypt(smsHistoryBinding(credentialID, reference), encrypted)
}
