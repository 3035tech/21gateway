package services

import (
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/forPelevin/gomoji"
	"golang.org/x/text/unicode/norm"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Keep the existing source length limit until its position in legacy processing is confirmed.
func prepareSMSText(text string) (string, apperrors.Code) {
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > MaximumMessageCharacters {
		return "", apperrors.CodeInvalidMessage
	}
	withoutEmoji := gomoji.RemoveEmojis(text)
	decomposed := norm.NFD.String(withoutEmoji)
	unaccented := strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, decomposed)
	return strings.TrimSpace(norm.NFC.String(unaccented)), apperrors.CodeNone
}
