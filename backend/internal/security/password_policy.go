package security

import (
	"strings"
	"unicode/utf8"
)

// Small offline denylist, not a breached-password database or entropy estimator.
// Existing API passwords and login verification are deliberately unaffected.
func ValidNewSystemPassword(password string) bool {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < MinimumSystemPasswordCharacters || len(password) > MaximumPasswordLength {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(password))
	if utf8.RuneCountInString(normalized) < MinimumSystemPasswordCharacters {
		return false
	}
	switch normalized {
	case "passwordpassword", "password123456789", "123456789012345", "12345678901234567890", "qwertyuiopasdfgh", "adminadminadmin", "senha1234567890", "21mobile21mobile", "letmeinletmeinletmein":
		return false
	}
	runes := []rune(normalized)
	if len(runes) == 0 {
		return false
	}
	for _, r := range runes[1:] {
		if r != runes[0] {
			return true
		}
	}
	return false
}
