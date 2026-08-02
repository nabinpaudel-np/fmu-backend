package mail

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"strings"

	gomail "github.com/go-mail/mail/v2"

	"fmu-backend/internal/config"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var (
	claimApprovedTmpl = template.Must(template.ParseFS(templateFS, "templates/claim_approved.html.tmpl"))
	passwordResetTmpl = template.Must(template.ParseFS(templateFS, "templates/password_reset.html.tmpl"))
)

// ClaimApprovedData feeds the claim-approved template.
type ClaimApprovedData struct {
	FullName        string
	Email           string
	Password        string
	InstitutionType string // "university" or "college"
	InstitutionName string
	Role            string
	LoginURL        string
	FromName        string // display name in the email header/footer; defaults to c.cfg.FromName
}

// PasswordResetData feeds the password-reset template.
type PasswordResetData struct {
	FullName      string
	Email         string
	ResetURL      string
	ExpiryMinutes int
	FromName      string // display name in the email header/footer; defaults to c.cfg.FromName
}

type Client struct {
	cfg config.MailConfig
}

// New returns a mailer configured from the project's MailConfig. Returns
// ErrNotConfigured when the SMTP credentials are missing so callers can
// decide whether to start without email.
func New(cfg config.MailConfig) (*Client, error) {
	if cfg.Username == "" || cfg.Password == "" {
		return nil, ErrNotConfigured
	}
	if cfg.From == "" {
		cfg.From = cfg.Username
	}
	if cfg.Server == "" {
		cfg.Server = "smtp.gmail.com"
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	return &Client{cfg: cfg}, nil
}

// SendClaimApproved renders the credentials email and ships it via SMTP.
// The institution type is lowercased and the role defaults to "representative"
// so the template grammar stays consistent across both claim types.
func (c *Client) SendClaimApproved(ctx context.Context, data ClaimApprovedData) error {
	if data.InstitutionType == "" {
		data.InstitutionType = "institution"
	} else {
		data.InstitutionType = strings.ToLower(data.InstitutionType)
	}
	if data.Role == "" {
		data.Role = "representative"
	}
	if data.FromName == "" {
		data.FromName = c.cfg.FromName
	}

	var buf bytes.Buffer
	if err := claimApprovedTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("%w: render: %v", ErrSendFailed, err)
	}

	m := gomail.NewMessage()
	if c.cfg.FromName != "" {
		m.SetAddressHeader("From", c.cfg.From, c.cfg.FromName)
	} else {
		m.SetHeader("From", c.cfg.From)
	}
	m.SetHeader("To", data.Email)
	m.SetHeader("Subject", fmt.Sprintf("Your %s claim has been approved", data.InstitutionType))
	m.SetBody("text/html", buf.String())

	dialer := gomail.NewDialer(c.cfg.Server, c.cfg.Port, c.cfg.Username, c.cfg.Password)
	dialer.StartTLSPolicy = startTLSPolicy(c.cfg.StartTLS, c.cfg.SSLTLS)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	client, err := dialer.Dial()
	if err != nil {
		return fmt.Errorf("%w: dial: %v", ErrSendFailed, err)
	}
	defer client.Close()

	if err := gomail.Send(client, m); err != nil {
		return fmt.Errorf("%w: send: %v", ErrSendFailed, err)
	}
	return nil
}

// SendPasswordReset renders the password-reset email and ships it via SMTP.
// ResetURL should be the absolute frontend link the user clicks (the token
// stays out of the email body — it's the URL query string). ExpiryMinutes
// is rounded into the body copy so the user knows the link's window.
func (c *Client) SendPasswordReset(ctx context.Context, data PasswordResetData) error {
	if data.FromName == "" {
		data.FromName = c.cfg.FromName
	}
	if data.ExpiryMinutes <= 0 {
		data.ExpiryMinutes = 60
	}

	var buf bytes.Buffer
	if err := passwordResetTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("%w: render: %v", ErrSendFailed, err)
	}

	m := gomail.NewMessage()
	if c.cfg.FromName != "" {
		m.SetAddressHeader("From", c.cfg.From, c.cfg.FromName)
	} else {
		m.SetHeader("From", c.cfg.From)
	}
	m.SetHeader("To", data.Email)
	m.SetHeader("Subject", "Reset your password")
	m.SetBody("text/html", buf.String())

	dialer := gomail.NewDialer(c.cfg.Server, c.cfg.Port, c.cfg.Username, c.cfg.Password)
	dialer.StartTLSPolicy = startTLSPolicy(c.cfg.StartTLS, c.cfg.SSLTLS)

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrSendFailed, err)
	}

	client, err := dialer.Dial()
	if err != nil {
		return fmt.Errorf("%w: dial: %v", ErrSendFailed, err)
	}
	defer client.Close()

	if err := gomail.Send(client, m); err != nil {
		return fmt.Errorf("%w: send: %v", ErrSendFailed, err)
	}
	return nil
}

// startTLSPolicy translates the project's STARTTLS/SSL_TLS booleans into
// the library's policy enum. The Gmail default (STARTTLS=true, SSL_TLS=false)
// maps to MandatoryStartTLS.
func startTLSPolicy(startTLS, sslTLS bool) gomail.StartTLSPolicy {
	if sslTLS {
		return gomail.NoStartTLS
	}
	if startTLS {
		return gomail.MandatoryStartTLS
	}
	return gomail.OpportunisticStartTLS
}
