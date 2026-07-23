package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"fmu-backend/internal/config"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/response"
	"fmu-backend/internal/token"
)

func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := GetAccessCookie(r)
			if raw == "" {
				response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
				return
			}

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

// OptionalAuthMiddleware runs the same JWT parse as AuthMiddleware but never
// rejects the request. If a valid cookie is present, claims are injected into
// the context; otherwise the handler proceeds anonymously. Use this on public
// endpoints that want to personalize responses (e.g. stamp `is_favorited` on
// list items) without forcing authentication.
func OptionalAuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := GetAccessCookie(r)
			if raw == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims := &token.AccessTokenClaims{}
			parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return []byte(cfg.AccessTokenSecret), nil
			})
			if err != nil || !parsed.Valid {
				next.ServeHTTP(w, r)
				return
			}

			ctx := WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireUniversityEditor gates a route so that only users who may edit a
// specific university can pass. The URL parameter `id` (or any name passed
// to idParam) is read as the target university's UUID.
//
// Admins are always allowed. Representatives are allowed only when their
// JWT-bound RepresentativeUniversityID matches the URL id. Anyone else gets
// 403.
func RequireUniversityEditor(idParam string) func(http.Handler) http.Handler {
	if idParam == "" {
		idParam = "id"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := ClaimsFromContext(r.Context())
			if err != nil {
				response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
				return
			}

			target := chi.URLParam(r, idParam)
			if claims.Role == RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}
			if claims.Role == RoleRepresentative && claims.RepresentativeUniversityID != "" && claims.RepresentativeUniversityID == target {
				next.ServeHTTP(w, r)
				return
			}
			response.Error(w, http.StatusForbidden, errs.ErrForbidden.Error())
		})
	}
}

// RequireCollegeEditor gates a route so that only users who may edit a
// specific college can pass. The URL parameter `id` (or any name passed to
// idParam) is read as the target college's UUID.
//
// Admins are always allowed. Representatives are allowed only when their
// JWT-bound RepresentativeCollegeID matches the URL id. Anyone else gets 403.
// A university-level representative whose id is NOT bound to a college is
// rejected — college editing requires the college-scoped representative role.
func RequireCollegeEditor(idParam string) func(http.Handler) http.Handler {
	if idParam == "" {
		idParam = "id"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := ClaimsFromContext(r.Context())
			if err != nil {
				response.Error(w, http.StatusUnauthorized, errs.ErrUnauthorized.Error())
				return
			}

			target := chi.URLParam(r, idParam)
			if claims.Role == RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}
			if claims.Role == RoleRepresentative && claims.RepresentativeCollegeID != "" && claims.RepresentativeCollegeID == target {
				next.ServeHTTP(w, r)
				return
			}
			response.Error(w, http.StatusForbidden, errs.ErrForbidden.Error())
		})
	}
}
