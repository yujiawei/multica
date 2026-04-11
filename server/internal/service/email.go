package service

import (
	"crypto/tls"
	"fmt"
	"html"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/resend/resend-go/v2"
)

// maxSubjectFieldRunes bounds how much user-controlled text (workspace name,
// inviter name) can land in an email Subject. Prevents attackers from stuffing
// a full phishing pitch into a workspace name that gets sent from our domain.
const maxSubjectFieldRunes = 60

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

// SendVerificationCode sends a one-time login code. The code is server-generated
// (6-digit numeric) so no user-controlled text reaches the email body here.
// If that ever changes, escape the user-controlled fields the same way
// SendInvitationEmail does.
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

// SendInvitationEmail notifies the invitee that they have been invited to a workspace.
// invitationID is included in the URL so the email deep-links to /invite/{id}.
func (s *EmailService) SendInvitationEmail(to, inviterName, workspaceName, invitationID string) error {
	appURL := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
	if appURL == "" {
		appURL = "https://app.multica.ai"
	}
	inviteURL := fmt.Sprintf("%s/invite/%s", appURL, invitationID)

	if s.client == nil {
		fmt.Printf("[DEV] Invitation email to %s: %s invited you to %s — %s\n", to, inviterName, workspaceName, inviteURL)
		return nil
	}

	params := buildInvitationParams(s.fromEmail, to, inviterName, workspaceName, inviteURL)
	_, err := s.client.Emails.Send(params)
	return err
}

// buildInvitationParams assembles the Resend request for an invitation email.
// Separated from SendInvitationEmail so the sanitization behavior is unit-testable
// without needing to mock the Resend SDK.
func buildInvitationParams(from, to, inviterName, workspaceName, inviteURL string) *resend.SendEmailRequest {
	safeWorkspace := html.EscapeString(workspaceName)
	safeInviter := html.EscapeString(inviterName)
	subjectInviter := sanitizeSubjectField(inviterName)
	subjectWorkspace := sanitizeSubjectField(workspaceName)

	return &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: fmt.Sprintf("%s invited you to %s on Multica", subjectInviter, subjectWorkspace),
		Html: fmt.Sprintf(
			`<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
				<h2>You're invited to join %s</h2>
				<p><strong>%s</strong> invited you to collaborate in the <strong>%s</strong> workspace on Multica.</p>
				<p style="margin: 24px 0;">
					<a href="%s" style="display: inline-block; padding: 12px 24px; background: #000; color: #fff; text-decoration: none; border-radius: 6px; font-weight: 500;">Accept invitation</a>
				</p>
				<p style="color: #666; font-size: 14px;">You'll need to log in to accept or decline the invitation.</p>
			</div>`, safeWorkspace, safeInviter, safeWorkspace, inviteURL),
	}
}

// sanitizeSubjectField prepares user-controlled text for the email Subject line.
// Subject is not HTML-rendered, so HTML-escaping would leak literal entities
// (e.g. &lt;script&gt;) into the recipient's inbox. Instead strip control
// characters (defense in depth against header-injection-adjacent abuse even
// though Resend also filters CR/LF) and cap length so attackers can't stuff
// a full phishing subject into a workspace name.
func sanitizeSubjectField(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	cleaned := b.String()
	if utf8.RuneCountInString(cleaned) <= maxSubjectFieldRunes {
		return cleaned
	}
	runes := []rune(cleaned)
	return string(runes[:maxSubjectFieldRunes-1]) + "…"
}
