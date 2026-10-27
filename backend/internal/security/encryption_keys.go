package security

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"regexp"
	"strings"
)

const versionedEnvelopePrefix = "v2:"

var encryptionKeyID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)

type encryptionKeys struct {
	legacy cipher.AEAD
	keys   map[string]cipher.AEAD
	active string
}

func newEncryptionKeys(legacy, active string, keys map[string]string) (*encryptionKeys, error) {
	result := &encryptionKeys{keys: make(map[string]cipher.AEAD), active: active}
	if legacy != "" {
		aead, err := decodeEncryptionKey(legacy)
		if err != nil {
			return nil, err
		}
		result.legacy = aead
	}
	for id, key := range keys {
		if !encryptionKeyID.MatchString(id) {
			return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
		aead, err := decodeEncryptionKey(key)
		if err != nil {
			return nil, err
		}
		result.keys[id] = aead
	}
	if active != "" && result.keys[active] == nil {
		return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	return result, nil
}
func decodeEncryptionKey(encoded string) (cipher.AEAD, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != providerKeyBytes {
		return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	return cipher.NewGCMWithRandomNonce(block)
}
func (k *encryptionKeys) enabled() bool { return k != nil && (k.active != "" || k.legacy != nil) }
func (k *encryptionKeys) encrypt(binding, plaintext string) (string, error) {
	if !k.enabled() {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	aead := k.legacy
	prefix := tokenEnvelopeVersion
	if k.active != "" {
		aead = k.keys[k.active]
		prefix = versionedEnvelopePrefix + k.active + ":"
	}
	return prefix + base64.RawStdEncoding.EncodeToString(aead.Seal(nil, nil, []byte(plaintext), []byte(binding))), nil
}
func (k *encryptionKeys) decrypt(binding, encrypted string) (string, error) {
	if k == nil {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	var aead cipher.AEAD
	data := ""
	if strings.HasPrefix(encrypted, tokenEnvelopeVersion) {
		aead = k.legacy
		data = strings.TrimPrefix(encrypted, tokenEnvelopeVersion)
	} else if strings.HasPrefix(encrypted, versionedEnvelopePrefix) {
		parts := strings.SplitN(strings.TrimPrefix(encrypted, versionedEnvelopePrefix), ":", 2)
		if len(parts) == 2 {
			aead = k.keys[parts[0]]
			data = parts[1]
		}
	}
	if aead == nil {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	raw, err := base64.RawStdEncoding.DecodeString(data)
	if err != nil {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	plain, err := aead.Open(nil, nil, raw, []byte(binding))
	if err != nil {
		return "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	return string(plain), nil
}
func (k *encryptionKeys) current(encrypted string) bool {
	if !k.enabled() {
		return false
	}
	if k.active != "" {
		return strings.HasPrefix(encrypted, versionedEnvelopePrefix+k.active+":")
	}
	return strings.HasPrefix(encrypted, tokenEnvelopeVersion)
}
