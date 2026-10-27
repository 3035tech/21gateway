package mail

import (
	"context"
	"crypto/tls"
	"net"
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
func (s *SMTP) send(ctx context.Context, to, token string, content messageContent) error {
	link := *s.AppURL
	link.Path = strings.TrimRight(link.Path, "/") + "/resetar-senha"
	link.Fragment = "token=" + token
	message, err := renderMessage(s.From, to, link.String(), content)
	if err != nil {
		return err
	}
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
