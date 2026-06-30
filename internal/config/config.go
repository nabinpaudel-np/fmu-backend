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
}

func Load() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
