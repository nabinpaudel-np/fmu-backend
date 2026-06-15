package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmu-backend/internal/config"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuthService interface {
	GetGoogleAuthURL() string
	ExchangeGoogleCode(ctx context.Context, code string) (*GoogleUser, error)
}

type oauthService struct {
	oauthConfig *oauth2.Config
}

func NewOAuthService(cfg *config.Config) OAuthService {
	return &oauthService{
		oauthConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

func (s *oauthService) GetGoogleAuthURL() string {
	return s.oauthConfig.AuthCodeURL("state")
}

func (s *oauthService) ExchangeGoogleCode(ctx context.Context, code string) (*GoogleUser, error) {
	token, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, errors.New("failed to exchange code with Google")
	}

	client := s.oauthConfig.Client(ctx, token)
	userResp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo?alt=json")
	if err != nil {
		return nil, err
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get user info from Google")
	}

	var googleUser GoogleUser
	if err := json.NewDecoder(userResp.Body).Decode(&googleUser); err != nil {
		return nil, err
	}

	return &googleUser, nil
}
