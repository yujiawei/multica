package service

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	// Resend mode
	resendClient *resend.Client
	// SMTP mode
	smtpServer   string
	smtpPort     string
	smtpUsername string
	smtpPassword string
	// Common
	fromEmail string
	mode      string // "resend", "smtp", or "dev"
}

func NewEmailService() *EmailService {
	svc := &EmailService{}

	// Check SMTP first (preferred when configured)
	smtpServer := os.Getenv("SMTP_SERVER")
	smtpUsername := os.Getenv("SMTP_USERNAME")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "465"
	}

	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = os.Getenv("RESEND_FROM_EMAIL")
	}
	if from == "" && smtpUsername != "" {
		from = smtpUsername
	}
	if from == "" {
		from = "noreply@multica.ai"
	}
	svc.fromEmail = from

	if smtpServer != "" && smtpUsername != "" && smtpPassword != "" {
		svc.smtpServer = smtpServer
		svc.smtpPort = smtpPort
		svc.smtpUsername = smtpUsername
		svc.smtpPassword = smtpPassword
		svc.mode = "smtp"
		fmt.Printf("[EMAIL] Using SMTP: %s:%s as %s\n", smtpServer, smtpPort, from)
		return svc
	}

	// Fall back to Resend
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey != "" {
		svc.resendClient = resend.NewClient(apiKey)
		svc.mode = "resend"
		fmt.Printf("[EMAIL] Using Resend API as %s\n", from)
		return svc
	}

	// Dev mode — print to log
	svc.mode = "dev"
	fmt.Println("[EMAIL] No email provider configured — codes will be printed to log")
	return svc
}

func (s *EmailService) SendVerificationCode(to, code string) error {
	subject := "Your Multica verification code"
	html := fmt.Sprintf(
		`<div style="font-family: sans-serif; max-width: 400px; margin: 0 auto;">
			<h2>Your verification code</h2>
			<p style="font-size: 32px; font-weight: bold; letter-spacing: 8px; margin: 24px 0;">%s</p>
			<p>This code expires in 10 minutes.</p>
			<p style="color: #666; font-size: 14px;">If you didn't request this code, you can safely ignore this email.</p>
		</div>`, code)

	switch s.mode {
	case "smtp":
		return s.sendSMTP(to, subject, html)
	case "resend":
		return s.sendResend(to, subject, html)
	default:
		fmt.Printf("[DEV] Verification code for %s: %s\n", to, code)
		return nil
	}
}

func (s *EmailService) sendResend(to, subject, html string) error {
	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}
	_, err := s.resendClient.Emails.Send(params)
	return err
}

func (s *EmailService) sendSMTP(to, subject, html string) error {
	addr := s.smtpServer + ":" + s.smtpPort

	headers := []string{
		"From: " + s.fromEmail,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=\"UTF-8\"",
	}
	msg := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + html)

	auth := smtp.PlainAuth("", s.smtpUsername, s.smtpPassword, s.smtpServer)

	// Port 465 uses implicit TLS (SMTPS)
	if s.smtpPort == "465" {
		return s.sendSMTPImplicitTLS(addr, auth, msg, to)
	}

	// Port 587 or others use STARTTLS
	return smtp.SendMail(addr, auth, s.fromEmail, []string{to}, msg)
}

func (s *EmailService) sendSMTPImplicitTLS(addr string, auth smtp.Auth, msg []byte, to string) error {
	tlsConfig := &tls.Config{
		ServerName: s.smtpServer,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.smtpServer)
	if err != nil {
		return fmt.Errorf("SMTP client failed: %w", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth failed: %w", err)
	}

	if err = client.Mail(s.fromEmail); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}

	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}

	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("SMTP write failed: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("SMTP close failed: %w", err)
	}

	return client.Quit()
}
