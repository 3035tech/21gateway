package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/21mobile/sms-gateway/internal/security"
)

type messageContent struct {
	subject string
	eyebrow string
	title   string
	intro   string
	action  string
	validity time.Duration
	note    string
}

var invitationMessage = messageContent{
	subject: "Seu convite para o 21 Mobile Gateway",
	eyebrow: "CONVITE DE ACESSO",
	title:   "Seu acesso começa aqui.",
	intro:   "Seu acesso ao painel 21 Mobile foi criado. Defina uma senha para entrar com segurança.",
	action:  "Definir minha senha",
	validity: security.InvitationLifetime,
	note:    "Se você não esperava este convite, pode ignorar esta mensagem.",
}

var passwordResetMessage = messageContent{
	subject: "Redefina sua senha no 21 Mobile Gateway",
	eyebrow: "SEGURANÇA DA CONTA",
	title:   "Vamos recuperar seu acesso.",
	intro:   "Recebemos um pedido para redefinir a senha da sua conta administrativa. Use o botão abaixo para escolher uma nova senha.",
	action:  "Redefinir senha",
	validity: security.PasswordResetLifetime,
	note:    "Se você não solicitou a redefinição, ignore esta mensagem. Sua senha atual continuará válida.",
}

var emailHTML = template.Must(template.New("email").Parse(`<!doctype html>
<html lang="pt-BR">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Subject}}</title></head>
<body style="margin:0;padding:0;background:#f4f1f7;color:#241b2e;font-family:Arial,Helvetica,sans-serif;-webkit-text-size-adjust:100%;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">{{.Preheader}}&nbsp;&zwnj;&nbsp;&zwnj;&nbsp;&zwnj;</div>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="width:100%;background:#f4f1f7;"><tr><td align="center" style="padding:28px 16px 36px;">
  <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="width:100%;max-width:600px;background:#ffffff;border:1px solid #e8e2ee;border-radius:18px;overflow:hidden;">
    <tr><td style="height:5px;background:#df147b;font-size:1px;line-height:5px;">&nbsp;</td></tr>
    <tr><td style="padding:28px 32px;background:#540064;">
      <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%"><tr>
        <td style="color:#ffffff;font-size:28px;font-weight:700;letter-spacing:-1.5px;line-height:1;">21<span style="font-size:19px;font-weight:400;letter-spacing:-1px;">MOBILE</span></td>
        <td align="right" style="color:#ead4ee;font-size:11px;font-weight:700;letter-spacing:2px;">GATEWAY</td>
      </tr></table>
    </td></tr>
    <tr><td style="padding:38px 32px 32px;">
      <p style="margin:0 0 14px;color:#79006e;font-size:11px;font-weight:700;letter-spacing:2px;line-height:1.5;">{{.Eyebrow}}</p>
      <h1 style="margin:0 0 18px;color:#241b2e;font-size:30px;font-weight:700;letter-spacing:-0.8px;line-height:1.18;">{{.Title}}</h1>
      <p style="margin:0 0 26px;color:#574e60;font-size:16px;line-height:1.65;">{{.Intro}}</p>
      <table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr><td bgcolor="#720078" style="border-radius:9px;background:#720078;">
        <a href="{{.Link}}" style="display:inline-block;padding:16px 25px;color:#ffffff;font-size:15px;font-weight:700;line-height:1.2;text-decoration:none;">{{.Action}} &rarr;</a>
      </td></tr></table>
      <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="margin-top:30px;background:#f8f4fa;border:1px solid #eee5f1;border-radius:10px;"><tr><td style="padding:17px 19px;">
        <p style="margin:0 0 5px;color:#540064;font-size:13px;font-weight:700;line-height:1.4;">Link pessoal e temporário</p>
        <p style="margin:0;color:#5f5667;font-size:13px;line-height:1.55;">Expira em {{.Validity}} e pode ser usado uma única vez.</p>
      </td></tr></table>
      <p style="margin:25px 0 7px;color:#5f5667;font-size:13px;line-height:1.55;">Se o botão não abrir, copie este endereço no navegador:</p>
      <p style="margin:0;word-break:break-all;color:#720078;font-size:12px;line-height:1.65;"><a href="{{.Link}}" style="color:#720078;text-decoration:underline;">{{.Link}}</a></p>
    </td></tr>
    <tr><td style="padding:20px 32px 25px;border-top:1px solid #eee8f1;background:#fcfafc;">
      <p style="margin:0;color:#6d6474;font-size:13px;line-height:1.55;">{{.Note}}</p>
    </td></tr>
  </table>
  <p style="margin:20px 0 0;color:#857a8d;font-size:11px;line-height:1.5;">21 Mobile Gateway &middot; Painel de SMS</p>
</td></tr></table>
</body></html>`))

func renderEmailHTML(link string, content messageContent) (string, error) {
	var output bytes.Buffer
	err := emailHTML.Execute(&output, struct {
		Subject, Preheader, Eyebrow, Title, Intro, Action, Validity, Note, Link string
	}{content.subject, content.intro, content.eyebrow, content.title, content.intro, content.action, validityLabel(content.validity), content.note, link})
	return output.String(), err
}

func validityLabel(duration time.Duration) string {
	if duration < time.Hour {
		return fmt.Sprintf("%d minutos", duration/time.Minute)
	}
	return fmt.Sprintf("%d horas", duration/time.Hour)
}

func renderMessage(from, to, link string, content messageContent) ([]byte, error) {
	if strings.ContainsAny(from+to, "\r\n") {
		return nil, fmt.Errorf("invalid email header")
	}
	if _, err := mail.ParseAddress(from); err != nil {
		return nil, err
	}
	if _, err := mail.ParseAddress(to); err != nil {
		return nil, err
	}
	html, err := renderEmailHTML(link, content)
	if err != nil {
		return nil, err
	}
	plain := fmt.Sprintf("%s\n\n%s\n\n%s: %s\n\nO link expira em %s e pode ser usado uma única vez.\n\n%s\n", content.title, content.intro, content.action, link, validityLabel(content.validity), content.note)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, part := range []struct{ kind, text string }{{"text/plain", plain}, {"text/html", html}} {
		header := textproto.MIMEHeader{}
		header.Set("Content-Type", part.kind+"; charset=UTF-8")
		header.Set("Content-Transfer-Encoding", "quoted-printable")
		section, err := writer.CreatePart(header)
		if err != nil {
			return nil, err
		}
		encoded := quotedprintable.NewWriter(section)
		if _, err := encoded.Write([]byte(part.text)); err != nil {
			return nil, err
		}
		if err := encoded.Close(); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	fmt.Fprintf(&output, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n", from, to, mime.QEncoding.Encode("utf-8", content.subject), writer.Boundary())
	output.Write(body.Bytes())
	return output.Bytes(), nil
}
