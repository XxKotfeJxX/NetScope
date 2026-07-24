package monitoring

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const (
	SMTPTLSStartTLS = "starttls"
	SMTPTLSImplicit = "tls"
	SMTPTLSNone     = "none"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLSMode  string
	Timeout  time.Duration
}

type SMTPSender struct {
	config SMTPConfig
}

func NewSMTPSender(config SMTPConfig) *SMTPSender {
	if config.Port == 0 {
		config.Port = 587
	}
	if config.TLSMode == "" {
		config.TLSMode = SMTPTLSStartTLS
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	return &SMTPSender{config: config}
}

func (s *SMTPSender) Deliver(
	ctx context.Context,
	destination string,
	alert Alert,
) error {
	recipient, err := mail.ParseAddress(destination)
	if err != nil {
		return fmt.Errorf("parse recipient: %w", err)
	}
	sender, err := mail.ParseAddress(s.config.From)
	if err != nil {
		return fmt.Errorf("parse sender: %w", err)
	}
	message := emailMessage(sender, recipient, alert)

	address := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	dialer := net.Dialer{Timeout: s.config.Timeout}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer connection.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = connection.SetDeadline(deadline)
	} else {
		_ = connection.SetDeadline(time.Now().Add(s.config.Timeout))
	}

	tlsConfig := &tls.Config{
		ServerName: s.config.Host,
		MinVersion: tls.VersionTLS12,
	}
	if s.config.TLSMode == SMTPTLSImplicit {
		tlsConnection := tls.Client(connection, tlsConfig)
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("establish SMTP TLS connection: %w", err)
		}
		connection = tlsConnection
	}

	client, err := smtp.NewClient(connection, s.config.Host)
	if err != nil {
		return fmt.Errorf("initialize SMTP client: %w", err)
	}
	defer client.Close()
	if s.config.TLSMode == SMTPTLSStartTLS {
		supported, _ := client.Extension("STARTTLS")
		if !supported {
			return errors.New("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if s.config.Username != "" {
		auth := smtp.PlainAuth(
			"",
			s.config.Username,
			s.config.Password,
			s.config.Host,
		)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate to SMTP server: %w", err)
		}
	}
	if err := client.Mail(sender.Address); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message: %w", err)
	}
	if _, err := io.WriteString(writer, message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("close SMTP session: %w", err)
	}
	return nil
}

func emailMessage(from *mail.Address, to *mail.Address, alert Alert) string {
	targetName := sanitizeHeader(alert.Target.Name)
	status := sanitizeHeader(string(alert.Status))
	subject := fmt.Sprintf("[NetScope] %s is %s", targetName, status)
	body := fmt.Sprintf(
		"NetScope monitoring event\r\n\r\n"+
			"Target: %s\r\n"+
			"Address: %s\r\n"+
			"Status: %s\r\n"+
			"Message: %s\r\n"+
			"Observed at: %s\r\n",
		alert.Target.Name,
		alert.Target.Address,
		alert.Status,
		alert.Message,
		alert.SentAt.Format(time.RFC3339),
	)
	return strings.Join([]string{
		"Date: " + alert.SentAt.Format(time.RFC1123Z),
		"From: " + from.String(),
		"To: " + to.String(),
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
}

func sanitizeHeader(value string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(value)
}
