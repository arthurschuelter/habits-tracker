package auth

import (
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// Mailer sends a plain-text email.
type Mailer interface {
	Send(to, subject, body string) error
}

// SMTPMailer sends email through an SMTP server, using STARTTLS when the
// server offers it. net/smtp only sends credentials over TLS or to localhost.
type SMTPMailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (m SMTPMailer) Send(to, subject, body string) error {
	if strings.ContainsAny(to+m.From+subject, "\r\n") {
		return errors.New("email headers must not contain line breaks")
	}
	var auth smtp.Auth
	if m.Username != "" {
		auth = smtp.PlainAuth("", m.Username, m.Password, m.Host)
	}
	msg := "From: " + m.From + "\r\nTo: " + to + "\r\nSubject: " + subject +
		"\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" +
		strings.ReplaceAll(body, "\n", "\r\n")
	if err := smtp.SendMail(net.JoinHostPort(m.Host, m.Port), auth, m.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}
