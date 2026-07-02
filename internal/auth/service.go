package auth

import (
	"context"
	"fmu-backend/internal/config"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/oauth"
	"fmu-backend/internal/token"
	"fmu-backend/internal/user"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error)
	Login(ctx context.Context, req *LoginRequest, userAgent string) (*LoginResponse, error)
	Refresh(ctx context.Context, refreshToken string, userAgent string) (*RefreshResponse, error)
	GoogleLogin(ctx context.Context, code, codeVerifier string, userAgent string) (*LoginResponse, error)
	GetGoogleAuthURL(state string) string
	FrontendURL() string
	Me(ctx context.Context, userID string) (*MeResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}

type authService struct {
	userService  user.UserService
	tokenService token.TokenService
	oauthService oauth.OAuthService
	cfg          *config.Config
}

func NewAuthService(cfg *config.Config, userService user.UserService, tokenService token.TokenService, oauthService oauth.OAuthService) AuthService {
	return &authService{
		cfg:          cfg,
		userService:  userService,
		tokenService: tokenService,
		oauthService: oauthService,
	}
}

func (s *authService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	u, err := s.userService.Create(ctx, req.FullName, req.Email, req.Password, RoleStudent)
	if err != nil {
		return nil, err
	}

	return &RegisterResponse{
		UserID:    u.ID,
		FullName:  u.FullName,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}, nil
}

func (s *authService) Login(ctx context.Context, req *LoginRequest, userAgent string) (*LoginResponse, error) {
	existing, err := s.userService.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return nil, errs.ErrInvalidCredentials
	}

	if existing.Password == nil {
		return nil, errs.ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(*existing.Password), []byte(req.Password))
	if err != nil {
		return nil, errs.ErrInvalidCredentials
	}

	res, err := s.buildLoginResponse(ctx, existing.ID, existing.Email, existing.Role, existing.Avatar, existing.FullName, userAgent)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string, userAgent string) (*RefreshResponse, error) {
	userID, err := s.tokenService.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	tokenHash := token.HashRefreshToken(refreshToken)
	if err := s.tokenService.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return nil, errs.ErrInternalServer
	}

	u, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, errs.ErrUserNotFound
	}

	accessToken, err := s.tokenService.CreateAccessToken(u.ID, u.Email, u.Role)
	if err != nil {
		return nil, errs.ErrInternalServer
	}

	newRefreshToken, err := s.tokenService.CreateRefreshToken(ctx, u.ID, userAgent)
	if err != nil {
		return nil, errs.ErrInternalServer
	}

	avatar := ""
	if u.Avatar != nil {
		avatar = *u.Avatar
	}

	return &RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		UserID:       u.ID,
		FullName:     u.FullName,
		Email:        u.Email,
		Avatar:       avatar,
	}, nil
}

func (s *authService) Me(ctx context.Context, userID string) (*MeResponse, error) {
	u, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, errs.ErrUserNotFound
	}

	avatar := ""
	if u.Avatar != nil {
		avatar = *u.Avatar
	}

	return &MeResponse{
		UserID:   u.ID,
		FullName: u.FullName,
		Email:    u.Email,
		Avatar:   avatar,
		Role:     u.Role,
	}, nil
}

func (s *authService) GetGoogleAuthURL(state string) string {
	return s.oauthService.GetGoogleAuthURL(state)
}

func (s *authService) FrontendURL() string {
	return s.cfg.FrontendURL
}

func (s *authService) GoogleLogin(ctx context.Context, code, codeVerifier string, userAgent string) (*LoginResponse, error) {
	googleUser, err := s.oauthService.ExchangeGoogleCode(ctx, code, codeVerifier)
	if err != nil {
		return nil, err
	}

	user, err := s.userService.GetByProvider(ctx, "google", googleUser.ID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		existingByEmail, err := s.userService.GetByEmail(ctx, googleUser.Email)
		if err != nil {
			return nil, err
		}
		if existingByEmail != nil {
			return nil, errs.ErrEmailAlreadyRegistered
		}

		user, err = s.userService.CreateWithOAuth(ctx, googleUser.Name, googleUser.Email, "google", googleUser.ID, googleUser.Picture)
		if err != nil {
			return nil, err
		}
	}

	res, err := s.buildLoginResponse(ctx, user.ID, user.Email, user.Role, user.Avatar, user.FullName, userAgent)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *authService) buildLoginResponse(ctx context.Context, userID, email, role string, avatar *string, fullName, userAgent string) (*LoginResponse, error) {
	accessToken, err := s.tokenService.CreateAccessToken(userID, email, role)
	if err != nil {
		return nil, errs.ErrInternalServer
	}

	refreshToken, err := s.tokenService.CreateRefreshToken(ctx, userID, userAgent)
	if err != nil {
		return nil, errs.ErrInternalServer
	}

	avatarStr := ""
	if avatar != nil {
		avatarStr = *avatar
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserID:       userID,
		FullName:     fullName,
		Email:        email,
		Avatar:       avatarStr,
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	if _, err := s.tokenService.ValidateRefreshToken(ctx, refreshToken); err != nil {
		return err
	}

	tokenHash := token.HashRefreshToken(refreshToken)
	return s.tokenService.DeleteByTokenHash(ctx, tokenHash)
}
