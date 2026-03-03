package resendclient

import (
	"context"
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/resend/resend-go/v2"
)

// Compile-time interface check.
var _ ports.EmailSender = (*Client)(nil)

// Client sends emails via the Resend API.
type Client struct {
	client *resend.Client
	from   string
}

// NewClient creates a new Resend email client.
func NewClient(apiKey, from string) *Client {
	return &Client{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

// SendVerificationEmail sends an email verification link to the user.
func (c *Client) SendVerificationEmail(_ context.Context, to, name, verificationURL string) error {
	subject := "Verify your Zenith account"
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 560px; margin: 0 auto; padding: 40px 20px; color: #e5e5e5; background-color: #0a0a0a;">
  <div style="text-align: center; margin-bottom: 32px;">
    <h1 style="color: #10b981; font-size: 24px; margin: 0;">Zenith</h1>
  </div>
  <div style="background-color: #171717; border: 1px solid #262626; border-radius: 12px; padding: 32px;">
    <h2 style="color: #fafafa; font-size: 20px; margin: 0 0 16px;">Verify your email</h2>
    <p style="color: #a3a3a3; font-size: 14px; line-height: 1.6; margin: 0 0 24px;">
      Hi %s, thanks for signing up for Zenith. Please verify your email address by clicking the button below.
    </p>
    <div style="text-align: center; margin: 24px 0;">
      <a href="%s" style="display: inline-block; background-color: #10b981; color: #ffffff; text-decoration: none; font-weight: 600; font-size: 14px; padding: 12px 32px; border-radius: 8px;">
        Verify Email
      </a>
    </div>
    <p style="color: #737373; font-size: 12px; line-height: 1.5; margin: 24px 0 0;">
      If you didn't create an account, you can safely ignore this email. This link expires in 24 hours.
    </p>
  </div>
</body>
</html>`, name, verificationURL)

	_, err := c.client.Emails.Send(&resend.SendEmailRequest{
		From:    c.from,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	})
	if err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}
	return nil
}
