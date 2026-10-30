package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

const dialTimeout = 5 * time.Second
const sendTimeout = 10 * time.Second

type SMTP struct {
	Host, Port, Username, Password, From string
	AppURL                               *url.URL
}

// Probe checks the same encrypted authentication path used for delivery without sending mail.
func (s *SMTP) Probe(ctx context.Context) error {
	conn, err := (&net.Dialer{Timeout: dialTimeout}).DialContext(ctx, "tcp", net.JoinHostPort(s.Host, s.Port))
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(sendTimeout)
	if bound, ok := ctx.Deadline(); ok && bound.Before(deadline) {
		deadline = bound
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return err
	}
	if err := client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
		return err
	}
	return client.Quit()
}

func (s *SMTP) SendPasswordReset(ctx context.Context, to, token string) error {
	return s.send(ctx, to, token, passwordResetMessage)
}
func (s *SMTP) SendInvitation(ctx context.Context, to, token string) error {
	return s.send(ctx, to, token, invitationMessage)
}
func (s *SMTP) SendCriticalAlert(ctx context.Context, to, title, description, path string) error {
	if _, err := mail.ParseAddress(to); err != nil {
		return err
	}
	if strings.ContainsAny(to+title+description, "\r\n") || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return fmt.Errorf("invalid alert message")
	}
	link := *s.AppURL
	link.Path = strings.TrimRight(link.Path, "/") + path
	var body strings.Builder
	fmt.Fprintf(&body, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", s.From, to, mime.QEncoding.Encode("utf-8", "21 Mobile: "+title))
	encoded := quotedprintable.NewWriter(&body)
	_, err := fmt.Fprintf(encoded, "%s\n\n%s\n\nAcesse o painel: %s\n", title, description, link.String())
	if err != nil {
		return err
	}
	if err := encoded.Close(); err != nil {
		return err
	}
	return s.sendRaw(ctx, to, []byte(body.String()))
}
func (s *SMTP) send(ctx context.Context, to, token string, content messageContent) error {
	link := *s.AppURL
	link.Path = strings.TrimRight(link.Path, "/") + "/resetar-senha"
	link.Fragment = "token=" + token
	message, err := renderMessage(s.From, to, link.String(), content)
	if err != nil {
		return err
	}
	return s.sendRaw(ctx, to, message)
}
func (s *SMTP) sendRaw(ctx context.Context, to string, message []byte) error {
	address := net.JoinHostPort(s.Host, s.Port)
	conn, err := (&net.Dialer{Timeout: dialTimeout}).DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(sendTimeout)); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return err
	}
	if err := client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
		return err
	}
	if err := client.Mail(s.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
