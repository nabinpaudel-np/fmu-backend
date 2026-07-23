package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AccessTokenClaims carries everything middlewares need to authorize a
// request without a DB hit. RepresentativeUniversityID and
// RepresentativeCollegeID are non-empty only when Role == "representative"
// — that's how RequireUniversityEditor and RequireCollegeEditor do their
// O(1) per-target checks.
type AccessTokenClaims struct {
	UserID                     string `json:"user_id"`
	Email                      string `json:"email"`
	Role                       string `json:"role"`
	RepresentativeUniversityID string `json:"rep_uni_id,omitempty"`
	RepresentativeCollegeID    string `json:"rep_col_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateAccessToken mints a signed JWT for the given user. Pass empty
// strings for repUniversityID / repCollegeID for non-representative users.
func GenerateAccessToken(userID, email, role, repUniversityID, repCollegeID, secretKey string, expiry time.Duration) (string, error) {
	claims := &AccessTokenClaims{
		UserID:                     userID,
		Email:                      email,
		Role:                       role,
		RepresentativeUniversityID: repUniversityID,
		RepresentativeCollegeID:    repCollegeID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenStr, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}
