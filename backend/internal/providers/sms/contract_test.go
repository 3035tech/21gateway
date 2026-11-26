package sms

import (
	"strings"
	"testing"

	"github.com/21mobile/sms-gateway/internal/models"
)

func TestSelectEncodingSupportsGSM7Latin1AndUCS2(t *testing.T) {
	for _, tc := range []struct {
		message string
		want    models.SMSEncoding
	}{
		{"plain GSM7", models.SMSEncodingGSM7},
		{"×", models.SMSEncodingLatin1},
		{"漢", models.SMSEncodingUCS2},
	} {
		t.Run(tc.message, func(t *testing.T) {
			if got := SelectEncoding(tc.message); got != tc.want {
				t.Fatalf("encoding %q; want %q", got, tc.want)
			}
		})
	}
}

func TestSelectBatchEncodingUsesOneEncodingForEveryItem(t *testing.T) {
	for _, tc := range []struct {
		name     string
		messages []string
		want     models.SMSEncoding
	}{
		{"GSM7", []string{"plain", "^"}, models.SMSEncodingGSM7},
		{"Latin1", []string{"plain", "×"}, models.SMSEncodingLatin1},
		{"UCS2", []string{"×", "漢"}, models.SMSEncodingUCS2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := SelectBatchEncoding(tc.messages); got != tc.want {
				t.Fatalf("encoding %q; want %q", got, tc.want)
			}
		})
	}
}

func TestProviderCharacterLimits(t *testing.T) {
	for _, tc := range []struct {
		name     string
		message  string
		encoding models.SMSEncoding
		want     bool
	}{
		{"GSM7 at limit", strings.Repeat("a", MaximumGSM7Characters), models.SMSEncodingGSM7, true},
		{"GSM7 over limit", strings.Repeat("a", MaximumGSM7Characters+1), models.SMSEncodingGSM7, false},
		{"GSM7 extension at limit", strings.Repeat("^", MaximumGSM7Characters/2), models.SMSEncodingGSM7, true},
		{"GSM7 extension over limit", strings.Repeat("^", MaximumGSM7Characters/2+1), models.SMSEncodingGSM7, false},
		{"Latin1 at limit", strings.Repeat("×", MaximumLatin1Characters), models.SMSEncodingLatin1, true},
		{"Latin1 over limit", strings.Repeat("×", MaximumLatin1Characters+1), models.SMSEncodingLatin1, false},
		{"UCS2 at limit", strings.Repeat("漢", MaximumUCS2Characters), models.SMSEncodingUCS2, true},
		{"UCS2 over limit", strings.Repeat("漢", MaximumUCS2Characters+1), models.SMSEncodingUCS2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FitsProviderMessageLength(tc.message, tc.encoding); got != tc.want {
				t.Fatalf("length validation %v; want %v", got, tc.want)
			}
		})
	}
}
