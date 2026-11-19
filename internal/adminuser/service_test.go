package adminuser

import (
	"context"
	"errors"
	"testing"

	"fmu-backend/internal/auth"
	"fmu-backend/internal/db/sqlc"
	"fmu-backend/internal/errs"
	"fmu-backend/internal/pagination"
	"fmu-backend/internal/passwordreset"
	"fmu-backend/internal/user"
)

// fakeRepo is a stand-in for the repository. It records what was asked of it
// so a test can check the service asked the right question.
type fakeRepo struct {
	rows []sqlc.ListUsersRow
	// admins is what CountAdmins reports.
	admins int64
	// count is what Count reports.
	count int64
	// detail is what GetByID returns.
	detail *sqlc.GetAdminUserByIDRow

	deletedID    string
	updatedID    string
	updatedRole  string
	updatedUniv  string
	updatedCol   string
	invalidated  string
	deletedCalls int
}

func (r *fakeRepo) List(ctx context.Context, role, search string, limit, offset int32) ([]sqlc.ListUsersRow, error) {
	return r.rows, nil
}

func (r *fakeRepo) Count(ctx context.Context, role, search string) (int64, error) {
	return r.count, nil
}

func (r *fakeRepo) GetByID(ctx context.Context, id string) (*sqlc.GetAdminUserByIDRow, error) {
	return r.detail, nil
}

func (r *fakeRepo) CountAdmins(ctx context.Context) (int64, error) {
	return r.admins, nil
}

func (r *fakeRepo) UpdateRole(ctx context.Context, id, role string, universityID, collegeID string) (*sqlc.UpdateUserRoleRow, error) {
	r.updatedID = id
	r.updatedRole = role
	r.updatedUniv = universityID
	r.updatedCol = collegeID
	row := sqlc.UpdateUserRoleRow{ID: id, Role: role, Email: "u@example.com", FullName: "U"}
	return &row, nil
}

func (r *fakeRepo) Delete(ctx context.Context, id string) error {
	r.deletedID = id
	r.deletedCalls++
	return nil
}

func (r *fakeRepo) InvalidateResetTokens(ctx context.Context, userID string) error {
	r.invalidated = userID
	return nil
}

// fakeUsers holds the users the service can look up.
type fakeUsers struct {
	byID map[string]*user.User
}

func (u *fakeUsers) GetByID(ctx context.Context, id string) (*user.User, error) {
	return u.byID[id], nil
}

// fakeReset records the forgot-password call.
type fakeReset struct {
	called bool
	email  string
	result *passwordreset.ForgotPasswordResponse
}

func (r *fakeReset) ForgotPassword(ctx context.Context, req *passwordreset.ForgotPasswordRequest) (*passwordreset.ForgotPasswordResponse, error) {
	r.called = true
	r.email = req.Email
	if r.result != nil {
		return r.result, nil
	}
	return &passwordreset.ForgotPasswordResponse{AccountType: passwordreset.AccountTypePassword}, nil
}

func newTestService(repo *fakeRepo, users *fakeUsers, reset *fakeReset) Service {
	return NewService(repo, users, reset)
}

// An admin cannot delete their own account.
func TestDelete_RefusesSelf(t *testing.T) {
	repo := &fakeRepo{admins: 3}
	users := &fakeUsers{byID: map[string]*user.User{"admin1": testUser("admin1", auth.RoleAdmin, true)}}
	svc := newTestService(repo, users, &fakeReset{})

	err := svc.Delete(context.Background(), "admin1", "admin1")
	if !errors.Is(err, errs.ErrCannotDeleteSelf) {
		t.Errorf("Delete of own account = %v, want ErrCannotDeleteSelf", err)
	}
	if repo.deletedCalls != 0 {
		t.Error("the account was deleted despite the refusal")
	}
}

// The last admin cannot be deleted — that would leave nobody able to
// manage the catalogue.
func TestDelete_RefusesLastAdmin(t *testing.T) {
	repo := &fakeRepo{admins: 1}
	users := &fakeUsers{byID: map[string]*user.User{"admin1": testUser("admin1", auth.RoleAdmin, true)}}
	svc := newTestService(repo, users, &fakeReset{})

	err := svc.Delete(context.Background(), "admin1", "admin2")
	if !errors.Is(err, errs.ErrCannotRemoveLastAdmin) {
		t.Errorf("Delete of last admin = %v, want ErrCannotRemoveLastAdmin", err)
	}
	if repo.deletedCalls != 0 {
		t.Error("the account was deleted despite the refusal")
	}
}

// A student is not protected by the admin count, so they can be removed.
func TestDelete_RemovesAStudent(t *testing.T) {
	repo := &fakeRepo{admins: 2}
	users := &fakeUsers{byID: map[string]*user.User{"s1": testUser("s1", auth.RoleStudent, true)}}
	svc := newTestService(repo, users, &fakeReset{})

	if err := svc.Delete(context.Background(), "s1", "admin1"); err != nil {
		t.Fatalf("Delete = %v, want no error", err)
	}
	if repo.deletedID != "s1" {
		t.Errorf("deleted %q, want s1", repo.deletedID)
	}
}

// The last admin cannot be demoted either.
func TestUpdateRole_RefusesDemotingTheLastAdmin(t *testing.T) {
	repo := &fakeRepo{admins: 1}
	users := &fakeUsers{byID: map[string]*user.User{"admin1": testUser("admin1", auth.RoleAdmin, true)}}
	svc := newTestService(repo, users, &fakeReset{})

	_, err := svc.UpdateRole(context.Background(), "admin1", &UpdateRoleRequest{Role: auth.RoleStudent})
	if !errors.Is(err, errs.ErrCannotRemoveLastAdmin) {
		t.Errorf("demoting the last admin = %v, want ErrCannotRemoveLastAdmin", err)
	}
}

// An unknown role is refused rather than written to the database.
func TestUpdateRole_RefusesUnknownRole(t *testing.T) {
	repo := &fakeRepo{admins: 2}
	users := &fakeUsers{byID: map[string]*user.User{"s1": testUser("s1", auth.RoleStudent, true)}}
	svc := newTestService(repo, users, &fakeReset{})

	_, err := svc.UpdateRole(context.Background(), "s1", &UpdateRoleRequest{Role: "superuser"})
	if !errors.Is(err, errs.ErrInvalidRole) {
		t.Errorf("unknown role = %v, want ErrInvalidRole", err)
	}
}

func testUser(id, role string, withPassword bool) *user.User {
	u := &user.User{ID: id, Role: role, Email: id + "@example.com", FullName: "User " + id}
	if withPassword {
		hash := "hash"
		u.Password = &hash
	}
	return u
}

// A representative must be bound to exactly one institution.
func TestUpdateRole_RepresentativeNeedsOneScope(t *testing.T) {
	repo := &fakeRepo{admins: 2}
	users := &fakeUsers{byID: map[string]*user.User{"s1": testUser("s1", auth.RoleStudent, true)}}
	svc := newTestService(repo, users, &fakeReset{})

	// Neither.
	_, err := svc.UpdateRole(context.Background(), "s1", &UpdateRoleRequest{Role: auth.RoleRepresentative})
	if !errors.Is(err, errs.ErrRepresentativeScopeRequired) {
		t.Errorf("representative with no scope = %v, want ErrRepresentativeScopeRequired", err)
	}

	// Both.
	_, err = svc.UpdateRole(context.Background(), "s1", &UpdateRoleRequest{
		Role:                       auth.RoleRepresentative,
		RepresentativeUniversityID: "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
		RepresentativeCollegeID:    "1b5f2c43-9f68-5d3f-ce74-3f8b9d5e0a21",
	})
	if !errors.Is(err, errs.ErrRepresentativeScopeRequired) {
		t.Errorf("representative with two scopes = %v, want ErrRepresentativeScopeRequired", err)
	}
}

// A representative keeps their binding; a student keeps none. A stale
// binding would keep granting edit rights over that institution.
func TestUpdateRole_ClearsTheScopeForOtherRoles(t *testing.T) {
	repo := &fakeRepo{admins: 2}
	users := &fakeUsers{byID: map[string]*user.User{"s1": testUser("s1", auth.RoleStudent, true)}}
	svc := newTestService(repo, users, &fakeReset{})

	_, err := svc.UpdateRole(context.Background(), "s1", &UpdateRoleRequest{
		Role:                       auth.RoleStudent,
		RepresentativeUniversityID: "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
	})
	if err != nil {
		t.Fatalf("UpdateRole = %v, want no error", err)
	}
	if repo.updatedUniv != "" || repo.updatedCol != "" {
		t.Error("the scope was kept for a role that has none")
	}
}

// A representative keeps exactly the binding they were given.
func TestUpdateRole_KeepsTheScopeForARepresentative(t *testing.T) {
	repo := &fakeRepo{admins: 2}
	users := &fakeUsers{byID: map[string]*user.User{"s1": testUser("s1", auth.RoleStudent, true)}}
	svc := newTestService(repo, users, &fakeReset{})

	_, err := svc.UpdateRole(context.Background(), "s1", &UpdateRoleRequest{
		Role:                       auth.RoleRepresentative,
		RepresentativeUniversityID: "9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10",
	})
	if err != nil {
		t.Fatalf("UpdateRole = %v, want no error", err)
	}
	if repo.updatedUniv == "" {
		t.Error("the university binding was dropped for a representative")
	}
	if repo.updatedCol != "" {
		t.Error("a college binding was set when only a university was given")
	}
}

// A Google-only account has no password, so a reset link would lead
// nowhere. The admin is told why instead.
func TestResetPassword_RefusesAnOAuthOnlyAccount(t *testing.T) {
	repo := &fakeRepo{}
	users := &fakeUsers{byID: map[string]*user.User{"s1": testUser("s1", auth.RoleStudent, false)}}
	reset := &fakeReset{}
	svc := newTestService(repo, users, reset)

	_, err := svc.ResetPassword(context.Background(), "s1")
	if !errors.Is(err, errs.ErrCannotResetOAuthOnlyPassword) {
		t.Errorf("reset of an OAuth-only account = %v, want ErrCannotResetOAuthOnlyPassword", err)
	}
	if reset.called {
		t.Error("the forgot-password flow ran for an account with no password")
	}
}

// The reset goes through the same forgot-password flow the public endpoint
// uses, and any earlier link is burned first.
func TestResetPassword_UsesTheForgotPasswordFlow(t *testing.T) {
	repo := &fakeRepo{}
	users := &fakeUsers{byID: map[string]*user.User{"s1": testUser("s1", auth.RoleStudent, true)}}
	reset := &fakeReset{}
	svc := newTestService(repo, users, reset)

	res, err := svc.ResetPassword(context.Background(), "s1")
	if err != nil {
		t.Fatalf("ResetPassword = %v, want no error", err)
	}
	if !reset.called {
		t.Error("the forgot-password flow was not called")
	}
	if reset.email != "s1@example.com" {
		t.Errorf("reset email = %q, want s1@example.com", reset.email)
	}
	if repo.invalidated != "s1" {
		t.Errorf("invalidated tokens for %q, want s1", repo.invalidated)
	}
	if !res.Sent {
		t.Error("the response says no email was sent")
	}
}

// A missing user is a 404, not a crash.
func TestGetByID_MissingUser(t *testing.T) {
	repo := &fakeRepo{}
	users := &fakeUsers{byID: map[string]*user.User{}}
	svc := newTestService(repo, users, &fakeReset{})

	_, err := svc.GetByID(context.Background(), "nope")
	if !errors.Is(err, errs.ErrUserNotFound) {
		t.Errorf("GetByID = %v, want ErrUserNotFound", err)
	}
}

// The list carries the name of the institution a representative is bound to,
// so the dashboard can show which one it is instead of a bare UUID.
func TestList_CarriesTheInstitutionName(t *testing.T) {
	name := "Kathmandu University"
	repo := &fakeRepo{
		rows: []sqlc.ListUsersRow{{
			ID:                           "r1",
			Email:                        "r1@example.com",
			FullName:                     "R",
			Role:                         auth.RoleRepresentative,
			RepresentativeUniversityName: &name,
		}},
		count: 1,
	}
	users := &fakeUsers{byID: map[string]*user.User{}}
	svc := newTestService(repo, users, &fakeReset{})

	items, _, err := svc.List(context.Background(), "", "", pagination.Query{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List = %v, want no error", err)
	}
	if items[0].RepresentativeUniversityName != "Kathmandu University" {
		t.Errorf("university name = %q, want Kathmandu University", items[0].RepresentativeUniversityName)
	}
}

// A representative with no institution — or one whose institution has been
// deleted — has no name, and the row still comes back.
func TestList_NoInstitutionNameIsNotAnError(t *testing.T) {
	repo := &fakeRepo{
		rows: []sqlc.ListUsersRow{{
			ID:       "r1",
			Email:    "r1@example.com",
			FullName: "R",
			Role:     auth.RoleRepresentative,
		}},
		count: 1,
	}
	users := &fakeUsers{byID: map[string]*user.User{}}
	svc := newTestService(repo, users, &fakeReset{})

	items, _, err := svc.List(context.Background(), "", "", pagination.Query{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List = %v, want no error", err)
	}
	if len(items) != 1 {
		t.Fatalf("List returned %d items, want 1", len(items))
	}
	if items[0].RepresentativeUniversityName != "" {
		t.Errorf("university name = %q, want empty", items[0].RepresentativeUniversityName)
	}
}

// The password hash never reaches the response.
func TestList_NeverCarriesThePassword(t *testing.T) {
	hash := "secret-hash"
	repo := &fakeRepo{
		rows:  []sqlc.ListUsersRow{{ID: "s1", Email: "s1@example.com", FullName: "S", Role: auth.RoleStudent, Password: &hash}},
		count: 1,
	}
	users := &fakeUsers{byID: map[string]*user.User{}}
	svc := newTestService(repo, users, &fakeReset{})

	items, total, err := svc.List(context.Background(), "", "", pagination.Query{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List = %v, want no error", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("List returned %d items, total %d, want 1 and 1", len(items), total)
	}
	if items[0].AccountType != "password" {
		t.Errorf("account type = %q, want password", items[0].AccountType)
	}
}

// An account with no password is marked as a Google account, so the
// dashboard can say why a reset would not work.
func TestList_MarksAnOAuthAccount(t *testing.T) {
	repo := &fakeRepo{
		rows:  []sqlc.ListUsersRow{{ID: "s1", Email: "s1@example.com", FullName: "S", Role: auth.RoleStudent}},
		count: 1,
	}
	users := &fakeUsers{byID: map[string]*user.User{}}
	svc := newTestService(repo, users, &fakeReset{})

	items, _, err := svc.List(context.Background(), "", "", pagination.Query{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List = %v, want no error", err)
	}
	if items[0].AccountType != "oauth" {
		t.Errorf("account type = %q, want oauth", items[0].AccountType)
	}
}
