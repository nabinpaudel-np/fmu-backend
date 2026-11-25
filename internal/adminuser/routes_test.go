package adminuser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"fmu-backend/internal/pagination"
)

// stubService answers every call without touching a database. It is only
// used to prove the routes are wired to a handler.
type stubService struct{}

func (stubService) List(ctx context.Context, role, search string, q pagination.Query) ([]UserListItem, int64, error) {
	return nil, 0, nil
}

func (stubService) GetByID(ctx context.Context, id string) (*UserDetailResponse, error) {
	return &UserDetailResponse{}, nil
}

func (stubService) UpdateRole(ctx context.Context, id string, req *UpdateRoleRequest) (*UserDetailResponse, error) {
	return &UserDetailResponse{}, nil
}

func (stubService) ResetPassword(ctx context.Context, id string) (*ResetPasswordResponse, error) {
	return &ResetPasswordResponse{}, nil
}

func (stubService) Delete(ctx context.Context, id, actingAdminID string) error {
	return nil
}

// The admin endpoints must all be mounted, and each one must be behind the
// admin role check — a student must not be able to reach them.
func TestRoutes_AreMountedAndGated(t *testing.T) {
	r := chi.NewRouter()

	var reached []string
	adminMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			reached = append(reached, req.URL.Path)
			next.ServeHTTP(w, req)
		})
	}
	authMW := func(next http.Handler) http.Handler { return next }

	RegisterRoutes(r, NewHandler(stubService{}), authMW, adminMW)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/users"},
		{http.MethodGet, "/api/v1/admin/users/abc"},
		{http.MethodPatch, "/api/v1/admin/users/abc/role"},
		{http.MethodPost, "/api/v1/admin/users/abc/reset-password"},
		{http.MethodDelete, "/api/v1/admin/users/abc"},
	}

	for _, tc := range cases {
		reached = nil
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if len(reached) != 1 || reached[0] != tc.path {
			t.Errorf("%s %s did not reach the handler (reached %v)", tc.method, tc.path, reached)
		}
	}
}

// The role check must run before the handler, so an admin-only route never
// reaches a non-admin.
func TestRoutes_AdminCheckRunsFirst(t *testing.T) {
	r := chi.NewRouter()

	var handlerRan bool
	adminMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
	}
	authMW := func(next http.Handler) http.Handler { return next }

	RegisterRoutes(r, NewHandler(stubService{}), authMW, adminMW)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if handlerRan {
		t.Error("the handler ran before the admin check")
	}
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}
