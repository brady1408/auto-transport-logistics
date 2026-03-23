package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
)

type Service struct {
	apiKey    string
	fromEmail string
}

func NewService(apiKey, fromEmail string) *Service {
	return &Service{apiKey: apiKey, fromEmail: fromEmail}
}

func (s *Service) Enabled() bool {
	return s.apiKey != ""
}

func (s *Service) SendPasswordReset(to, username, resetURL string) error {
	body := fmt.Sprintf(`<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
  <h2>Password Reset</h2>
  <p>Hi <strong>%s</strong>,</p>
  <p>We received a request to reset your password. Click the button below to choose a new password:</p>
  <p style="text-align: center; margin: 30px 0;">
    <a href="%s" style="background: #2563eb; color: #fff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600;">Reset Password</a>
  </p>
  <p style="font-size: 13px; color: #666;">This link expires in 1 hour.</p>
  <p style="font-size: 13px; color: #666;">If you didn't request this, you can safely ignore this email.</p>
</div>`, html.EscapeString(username), html.EscapeString(resetURL))

	return s.send(to, "ATLinks - Password Reset", body)
}

func (s *Service) SendVerification(to, companyName, verifyURL string) error {
	body := fmt.Sprintf(`<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
  <h2>Verify Your Email</h2>
  <p>Thanks for registering <strong>%s</strong> on ATLinks.</p>
  <p>Click the button below to verify your email and activate your company account:</p>
  <p style="text-align: center; margin: 30px 0;">
    <a href="%s" style="background: #2563eb; color: #fff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600;">Verify Email</a>
  </p>
  <p style="font-size: 13px; color: #666;">This link expires in 24 hours.</p>
  <p style="font-size: 13px; color: #666;">If you didn't register, you can safely ignore this email.</p>
</div>`, html.EscapeString(companyName), html.EscapeString(verifyURL))

	return s.send(to, "ATLinks - Verify Your Email", body)
}

func (s *Service) SendContactConfirmation(to, name string) error {
	body := fmt.Sprintf(`<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
  <h2>We Got Your Message</h2>
  <p>Hi <strong>%s</strong>,</p>
  <p>Thanks for reaching out! We've received your inquiry about Atlas Cloud and a member of our team will get back to you shortly.</p>
  <p style="font-size: 13px; color: #666; margin-top: 30px;">In the meantime, feel free to explore our platform at <a href="https://atlascloud.app" style="color: #2563eb;">atlascloud.app</a>.</p>
</div>`, html.EscapeString(name))

	return s.send(to, "Atlas Cloud - We Received Your Inquiry", body)
}

func (s *Service) SendContactNotification(name, contactEmail, phone, company, message string) error {
	phoneLine := ""
	if phone != "" {
		phoneLine = fmt.Sprintf(`<p><strong>Phone:</strong> %s</p>`, html.EscapeString(phone))
	}
	companyLine := ""
	if company != "" {
		companyLine = fmt.Sprintf(`<p><strong>Company:</strong> %s</p>`, html.EscapeString(company))
	}
	messageLine := ""
	if message != "" {
		messageLine = fmt.Sprintf(`<p><strong>Message:</strong> %s</p>`, html.EscapeString(message))
	}

	body := fmt.Sprintf(`<div style="font-family: sans-serif; max-width: 480px; margin: 0 auto;">
  <h2>New Contact Inquiry</h2>
  <p><strong>Name:</strong> %s</p>
  <p><strong>Email:</strong> %s</p>
  %s%s%s
</div>`, html.EscapeString(name), html.EscapeString(contactEmail), phoneLine, companyLine, messageLine)

	return s.send(s.fromEmail, "Atlas Cloud - New Contact Inquiry from "+name, body)
}

func (s *Service) send(to, subject, html string) error {
	if !s.Enabled() {
		return fmt.Errorf("email service not configured (no RESEND_API_KEY)")
	}

	payload := map[string]any{
		"from":    s.fromEmail,
		"to":      []string{to},
		"subject": subject,
		"html":    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, errBody.String())
	}

	return nil
}
