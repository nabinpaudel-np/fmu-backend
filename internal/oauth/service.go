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
	GetGoogleAuthURL(state string) string
	ExchangeGoogleCode(ctx context.Context, code, codeVerifier, redirectURI string) (*GoogleUser, error)
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

// GetGoogleAuthURL builds the Google auth URL. `state` doubles as both the
// OAuth state (CSRF) and PKCE code_verifier. `prompt=select_account` forces
// the picker so multi-account users don't get silently logged in as the
// wrong one.
func (s *oauthService) GetGoogleAuthURL(state string) string {
	return s.oauthConfig.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(state),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
}

func (s *oauthService) ExchangeGoogleCode(ctx context.Context, code, codeVerifier, redirectURI string) (*GoogleUser, error) {
	// RFC 6749 §4.1.3: the redirect_uri sent to the token endpoint MUST
	// match the one used in the authorization request. The SPA builds the
	// auth URL itself, so it sends its redirect_uri back with the code
	// and we echo it here. We construct a fresh config per request rather
	// than mutating the stored one so concurrent exchanges don't race.
	exchangeCfg := &oauth2.Config{
		ClientID:     s.oauthConfig.ClientID,
		ClientSecret: s.oauthConfig.ClientSecret,
		RedirectURL:  redirectURI,
		Scopes:       s.oauthConfig.Scopes,
		Endpoint:     s.oauthConfig.Endpoint,
	}

	token, err := exchangeCfg.Exchange(ctx, code, oauth2.VerifierOption(codeVerifier))
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
