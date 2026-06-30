package auth

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"fmu-backend/internal/config"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/response"
	"fmu-backend/internal/token"
)

func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
				return
			}

			raw := strings.TrimPrefix(header, "Bearer ")

			claims := &token.AccessTokenClaims{}
			parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return []byte(cfg.AccessTokenSecret), nil
			})
			if err != nil || !parsed.Valid {
				response.Error(w, http.StatusUnauthorized, errs.ErrInvalidToken.Error())
				return
			}

			ctx := WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := ClaimsFromContext(r.Context())
			if err != nil {
				response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				response.Error(w, http.StatusForbidden, errs.ErrForbidden.Error())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
