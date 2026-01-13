package providers

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/quickchat/notification-service/internal/config"
	"github.com/quickchat/notification-service/internal/models"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/sirupsen/logrus"
)

// EmailProvider implements email notifications
type EmailProvider struct {
	config config.EmailConfig
	log    *logrus.Logger
	client *sendgrid.Client
}

// NewEmailProvider creates a new email provider
func NewEmailProvider(cfg config.EmailConfig, log *logrus.Logger) (*EmailProvider, error) {
	provider := &EmailProvider{
		config: cfg,
		log:    log,
	}

	if cfg.Provider == "sendgrid" && cfg.APIKey != "" {
		provider.client = sendgrid.NewSendClient(cfg.APIKey)
	}

	return provider, nil
}

// Name returns the provider name
func (p *EmailProvider) Name() string {
	return "email"
}

// Send sends a notification
func (p *EmailProvider) Send(ctx context.Context, notification *models.Notification) error {
	email := &models.EmailNotification{
		Notification: *notification,
		ToEmail:      "", // Would need to be set from notification data
		Subject:      notification.Title,
		HTMLBody:     notification.Body,
	}
	return p.SendEmail(ctx, email)
}

// SendBatch sends multiple notifications
func (p *EmailProvider) SendBatch(ctx context.Context, notifications []*models.Notification) ([]SendResult, error) {
	results := make([]SendResult, len(notifications))
	for i, n := range notifications {
		err := p.Send(ctx, n)
		results[i] = SendResult{
			NotificationID: n.ID.Hex(),
			Success:        err == nil,
			Error:          err,
		}
	}
	return results, nil
}

// SendEmail sends an email notification
func (p *EmailProvider) SendEmail(ctx context.Context, email *models.EmailNotification) error {
	switch p.config.Provider {
	case "sendgrid":
		return p.sendViaSendGrid(ctx, email)
	case "smtp":
		return p.sendViaSMTP(ctx, email)
	default:
		return fmt.Errorf("unknown email provider: %s", p.config.Provider)
	}
}

// sendViaSendGrid sends email using SendGrid API
func (p *EmailProvider) sendViaSendGrid(ctx context.Context, email *models.EmailNotification) error {
	from := mail.NewEmail(p.config.FromName, p.config.FromEmail)
	to := mail.NewEmail(email.ToName, email.ToEmail)

	var message *mail.SGMailV3

	if email.TemplateID != "" {
		// Use dynamic template
		message = mail.NewV3Mail()
		message.SetFrom(from)
		message.SetTemplateID(email.TemplateID)

		personalization := mail.NewPersonalization()
		personalization.AddTos(to)

		// Add dynamic template data
		for k, v := range email.TemplateData {
			personalization.SetDynamicTemplateData(k, v)
		}

		message.AddPersonalizations(personalization)
	} else {
		// Regular email
		subject := email.Subject
		if subject == "" {
			subject = email.Title
		}

		htmlContent := email.HTMLBody
		if htmlContent == "" {
			htmlContent = fmt.Sprintf("<p>%s</p>", email.Body)
		}

		textContent := email.TextBody
		if textContent == "" {
			textContent = email.Body
		}

		message = mail.NewSingleEmail(from, subject, to, textContent, htmlContent)
	}

	// Add reply-to if specified
	if email.ReplyTo != "" {
		message.SetReplyTo(mail.NewEmail("", email.ReplyTo))
	}

	// Add attachments
	for _, attachment := range email.Attachments {
		sgAttachment := mail.NewAttachment()
		sgAttachment.SetFilename(attachment.Filename)
		sgAttachment.SetType(attachment.ContentType)
		sgAttachment.SetContent(string(attachment.Content))
		message.AddAttachment(sgAttachment)
	}

	response, err := p.client.SendWithContext(ctx, message)
	if err != nil {
		p.log.Errorf("SendGrid send failed: %v", err)
		return fmt.Errorf("SendGrid send failed: %w", err)
	}

	if response.StatusCode >= 400 {
		p.log.Errorf("SendGrid error: %d - %s", response.StatusCode, response.Body)
		return fmt.Errorf("SendGrid error: %d", response.StatusCode)
	}

	p.log.Debugf("Email sent via SendGrid to: %s", email.ToEmail)
	return nil
}

// sendViaSMTP sends email using SMTP
func (p *EmailProvider) sendViaSMTP(ctx context.Context, email *models.EmailNotification) error {
	// Build message
	subject := email.Subject
	if subject == "" {
		subject = email.Title
	}

	body := email.HTMLBody
	if body == "" {
		body = email.Body
	}

	message := fmt.Sprintf("From: %s <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s",
		p.config.FromName, p.config.FromEmail,
		email.ToEmail,
		subject,
		body,
	)

	// SMTP auth
	auth := smtp.PlainAuth("",
		p.config.SMTPUser,
		p.config.SMTPPass,
		p.config.SMTPHost,
	)

	// Send
	addr := fmt.Sprintf("%s:%d", p.config.SMTPHost, p.config.SMTPPort)
	err := smtp.SendMail(addr, auth, p.config.FromEmail, []string{email.ToEmail}, []byte(message))
	if err != nil {
		p.log.Errorf("SMTP send failed: %v", err)
		return fmt.Errorf("SMTP send failed: %w", err)
	}

	p.log.Debugf("Email sent via SMTP to: %s", email.ToEmail)
	return nil
}

// IsHealthy checks if the email provider is available
func (p *EmailProvider) IsHealthy(ctx context.Context) bool {
	return p.client != nil || (p.config.SMTPHost != "" && p.config.SMTPPort > 0)
}
