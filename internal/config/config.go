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

func Load() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
