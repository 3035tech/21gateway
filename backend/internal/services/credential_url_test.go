package services

import "testing"

func TestValidateCallbackURLRejectsReservedQueryParams(t *testing.T) {
	for _, raw := range []string{
		"https://client.example/dlr?idSms=1",
		"https://client.example/dlr?correlationId=abc",
		"https://client.example/dlr?status=DELIVERED",
		"https://client.example/dlr?tenant=foo&status=X",
	} {
		if ValidateCallbackURL(raw) {
			t.Errorf("expected rejection of URL with reserved param: %s", raw)
		}
	}
}

func TestValidateCallbackURLAcceptsValidURLs(t *testing.T) {
	for _, raw := range []string{
		"",
		"https://client.example/dlr",
		"https://client.example/dlr?tenant=foo",
		"https://client.example:443/dlr",
	} {
		if !ValidateCallbackURL(raw) {
			t.Errorf("expected acceptance of valid URL: %s", raw)
		}
	}
}
