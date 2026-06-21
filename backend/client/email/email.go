package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/subbu/family_tree/config"
)

type InviteMessage struct {
	ToEmail     string
	FamilyName  string
	Role        string
	SiteURL     string
	InviterName string
}

type Client interface {
	Enabled() bool
	SendInvite(ctx context.Context, msg InviteMessage) error
}

type client struct {
	host    string
	port    string
	user    string
	pass    string
	from    string
	siteURL string
	enabled bool
}

type noop struct{}

func New(cfg config.Config) Client {
	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" {
		return noop{}
	}
	port := cfg.SMTPPort
	if port == "" {
		port = "587"
	}
	return &client{
		host:    cfg.SMTPHost,
		port:    port,
		user:    cfg.SMTPUser,
		pass:    cfg.SMTPPassword,
		from:    cfg.SMTPFrom,
		siteURL: cfg.FrontendURL,
		enabled: true,
	}
}

func (noop) Enabled() bool { return false }

func (noop) SendInvite(context.Context, InviteMessage) error { return nil }

func (c *client) Enabled() bool { return c.enabled }

func (c *client) SendInvite(_ context.Context, msg InviteMessage) error {
	siteURL := strings.TrimRight(msg.SiteURL, "/")
	if siteURL == "" {
		siteURL = strings.TrimRight(c.siteURL, "/")
	}

	inviter := strings.TrimSpace(msg.InviterName)
	if inviter == "" {
		inviter = "Someone"
	}

	subject := fmt.Sprintf(`You're invited to "%s" on Family Tree`, msg.FamilyName)
	body := fmt.Sprintf(`Hello,

%s invited you to join the "%s" family tree as a %s.

To accept:
1. Open %s
2. Sign in with Google using this email address: %s
3. On the home page, click Accept on your pending invite

This invite expires in 7 days. You must use the same Google account as the email this was sent to.

— Family Tree
`, inviter, msg.FamilyName, msg.Role, siteURL, msg.ToEmail)

	return c.send(msg.ToEmail, subject, body)
}

func (c *client) send(to, subject, body string) error {
	addr := net.JoinHostPort(c.host, c.port)
	from := c.from

	var auth smtp.Auth
	if c.user != "" {
		auth = smtp.PlainAuth("", c.user, c.pass, c.host)
	}

	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}
	message := []byte(strings.Join(headers, "\r\n"))

	if c.port == "465" {
		return c.sendTLS(addr, auth, from, to, message)
	}
	return smtp.SendMail(addr, auth, extractEmail(from), []string{to}, message)
}

func (c *client) sendTLS(addr string, auth smtp.Auth, from, to string, message []byte) error {
	tlsConfig := &tls.Config{ServerName: c.host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	smtpClient, err := smtp.NewClient(conn, c.host)
	if err != nil {
		return err
	}
	defer smtpClient.Close()

	if auth != nil {
		if err := smtpClient.Auth(auth); err != nil {
			return err
		}
	}
	if err := smtpClient.Mail(extractEmail(from)); err != nil {
		return err
	}
	if err := smtpClient.Rcpt(to); err != nil {
		return err
	}
	writer, err := smtpClient.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return smtpClient.Quit()
}

func extractEmail(from string) string {
	if i := strings.Index(from, "<"); i >= 0 {
		if j := strings.Index(from, ">"); j > i {
			return strings.TrimSpace(from[i+1 : j])
		}
	}
	return strings.TrimSpace(from)
}