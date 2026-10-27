package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"html"
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

func (s *SMTP) SendPasswordReset(ctx context.Context, to, token string) error {
	return s.send(ctx, to, token, "Recuperar acesso", "Redefina sua senha", "Recebemos um pedido para redefinir a senha da sua conta administrativa.")
}
func (s *SMTP) SendInvitation(ctx context.Context, to, token string) error {
	return s.send(ctx, to, token, "Convite para o 21 Mobile Gateway", "Defina sua senha", "Seu acesso ao painel 21 Mobile Gateway foi criado. Defina sua senha para começar.")
}
func (s *SMTP) send(ctx context.Context, to, token, subject, title, intro string) error {
	link := *s.AppURL
	link.Path = strings.TrimRight(link.Path, "/") + "/resetar-senha"
	link.Fragment = "token=" + token
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
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<!doctype html><html lang=\"pt-BR\"><body style=\"margin:0;background:#f5f3f9;font-family:Arial,sans-serif;color:#24143d\"><div style=\"max-width:560px;margin:32px auto;background:#fff;border-radius:18px;overflow:hidden\"><div style=\"padding:28px;background:#3e1468;color:white;font-size:22px;font-weight:bold\">21 Mobile Gateway</div><div style=\"padding:32px\"><h1 style=\"font-size:25px;margin:0 0 16px\">%s</h1><p style=\"font-size:15px;line-height:1.6\">%s</p><a href=\"%s\" style=\"display:inline-block;margin:16px 0;padding:14px 24px;background:#ff0066;border-radius:10px;color:white;text-decoration:none;font-weight:bold\">Definir senha</a><p style=\"font-size:13px;line-height:1.6;color:#655b70\">O link expira em 15 minutos e funciona uma única vez. Se você não solicitou, ignore este e-mail.</p></div></div></body></html>", s.From, to, subject, html.EscapeString(title), html.EscapeString(intro), html.EscapeString(link.String()))
	if _, err := writer.Write([]byte(message)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
