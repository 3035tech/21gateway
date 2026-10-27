package errors

import "testing"

func TestPublicErrorCatalogsCoverKnownCodes(t *testing.T) {
	known := []Code{
		CodeInvalidRequest, CodeUnauthorized, CodeForbidden, CodeNotFound, CodeRateLimited,
		CodeServiceUnavailable, CodeInternal, CodeConfigurationInvalid, CodeDatabaseUnavailable,
		CodeInvalidDestination, CodeInvalidMessage, CodeBatchUnsupported, CodeScheduleUnsupported,
		CodeProviderUnauthorized, CodeInsufficientBalance, CodeProviderRejected, CodeUnknownOutcome,
		CodeConflict, CodeLastAdmin, CodeCSRFInvalid, CodeTestDestinationNotAllowed,
		CodeTestNotConfigured, CodeInvalidCurrentPassword, CodeMFARequired, CodeMFAInvalid,
		CodeInvalidResetToken, CodeInvitationDeliveryFailed,
	}
	for _, language := range []string{"pt-BR", "en-US"} {
		for _, code := range known {
			if messages[language][code] == "" {
				t.Fatalf("missing %s in %s", code, language)
			}
		}
	}
	if CodeUnauthorized.Message() != "Autenticação necessária." {
		t.Fatal("default legacy message changed")
	}
}
