package mail

import (
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
)

func TestEmailTemplatesIncludeAccessibleAlternatives(t *testing.T) {
	link := "https://gateway.example/resetar-senha#token=sample-token"
	for _, test := range []struct {
		name    string
		content messageContent
		button  string
		validity string
	}{
		{"invitation", invitationMessage, "Definir minha senha", "24 horas"},
		{"password reset", passwordResetMessage, "Redefinir senha", "15 minutos"},
	} {
		t.Run(test.name, func(t *testing.T) {
			message, err := renderMessage("gateway@example.com", "person@example.com", link, test.content)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := mail.ReadMessage(strings.NewReader(string(message)))
			if err != nil {
				t.Fatal(err)
			}
			_, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
			if err != nil {
				t.Fatal(err)
			}
			reader := multipart.NewReader(parsed.Body, params["boundary"])
			for _, expectedType := range []string{"text/plain", "text/html"} {
				part, err := reader.NextPart()
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(part.Header.Get("Content-Type"), expectedType) {
					t.Fatalf("got %q, want %q", part.Header.Get("Content-Type"), expectedType)
				}
				decoded, err := io.ReadAll(quotedprintable.NewReader(part))
				if err != nil {
					t.Fatal(err)
				}
				body := string(decoded)
				if !strings.Contains(body, link) || !strings.Contains(body, test.button) || !strings.Contains(body, test.validity) {
					t.Fatalf("missing action, link or expiration in %s part", expectedType)
				}
			}
		})
	}
}

func TestEmailTemplateEscapesLinkAndRejectsHeaderInjection(t *testing.T) {
	html, err := renderEmailHTML("https://gateway.example/#token=<unsafe>", invitationMessage)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<unsafe>") || !strings.Contains(html, "&lt;unsafe&gt;") {
		t.Fatal("link must be escaped in HTML")
	}
	if _, err := renderMessage("gateway@example.com\r\nBcc: stolen@example.com", "person@example.com", "https://gateway.example", invitationMessage); err == nil {
		t.Fatal("header injection was accepted")
	}
}
