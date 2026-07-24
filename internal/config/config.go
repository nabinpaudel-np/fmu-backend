package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnv string `env:"APP_ENV" envDefault:"development"`
	Port   string `env:"PORT" envDefault:"8080"`

	DBHost             string        `env:"DB_HOST,required"`
	DBUser             string        `env:"DB_USER,required"`
	DBPassword         string        `env:"DB_PASSWORD,required"`
	DBName             string        `env:"DB_NAME,required"`
	DBPort             int           `env:"DB_PORT" envDefault:"5432"`
	DatabaseURL        string        `env:"DATABASE_URL" envDefault:""`
	AccessTokenSecret  string        `env:"ACCESS_TOKEN_SECRET,required"`
	AccessTokenExpiry  time.Duration `env:"ACCESS_TOKEN_EXPIRY" envDefault:"60m"`
	RefreshTokenExpiry time.Duration `env:"REFRESH_TOKEN_EXPIRY" envDefault:"168h"`

	GoogleClientID     string `env:"GOOGLE_CLIENT_ID,required"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET,required"`
	GoogleRedirectURL  string `env:"GOOGLE_REDIRECT_URL,required"`

	AllowedOrigins string `env:"ALLOWED_ORIGINS" envDefault:""`

	CookieDomain   string `env:"COOKIE_DOMAIN" envDefault:""`
	CookieSecure   bool   `env:"COOKIE_SECURE" envDefault:"false"`
	CookieSameSite string `env:"COOKIE_SAME_SITE" envDefault:"lax"`

	FrontendURL string `env:"FRONTEND_URL" envDefault:"http://localhost:3000"`

	Cloudinary CloudinaryConfig `envPrefix:"CLOUDINARY_"`
	Supabase   SupabaseConfig   `envPrefix:"SUPABASE_"`
	Mail       MailConfig       `envPrefix:"MAIL_"`
}

type CloudinaryConfig struct {
	CloudName      string `env:"CLOUD_NAME"`
	APIKey         string `env:"API_KEY"`
	APISecret      string `env:"API_SECRET"`
	Folder         string `env:"FOLDER" envDefault:"fmu"`
	SecureDelivery bool   `env:"SECURE_DELIVERY" envDefault:"true"`
}

type SupabaseConfig struct {
	URL            string `env:"URL"`
	ServiceRoleKey string `env:"SERVICE_ROLE_KEY"`
	DocsBucket     string `env:"DOCS_BUCKET" envDefault:"documents"`
}

// MailConfig holds the SMTP settings. Username + Password are required;
// everything else has sane defaults that work for Gmail.
//
//	SMTP_USERNAME   smtp user (usually the From address)
//	SMTP_PASSWORD   smtp password (Gmail App Password for Gmail accounts)
//	SMTP_FROM       From address; falls back to Username if empty
//	SMTP_FROM_NAME  display name shown next to the From address
//	SMTP_SERVER     smtp host (default smtp.gmail.com)
//	SMTP_PORT       smtp port (default 587)
//	SMTP_STARTTLS   enable STARTTLS (default true)
//	SMTP_SSL_TLS    connect over implicit TLS / SMTPS (default false)
type MailConfig struct {
	Username  string `env:"USERNAME,required"`
	Password  string `env:"PASSWORD,required"`
	From      string `env:"FROM"`
	FromName  string `env:"FROM_NAME" envDefault:""`
	Server    string `env:"SERVER" envDefault:"smtp.gmail.com"`
	Port      int    `env:"PORT" envDefault:"587"`
	StartTLS  bool   `env:"STARTTLS" envDefault:"true"`
	SSLTLS    bool   `env:"SSL_TLS" envDefault:"false"`
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
