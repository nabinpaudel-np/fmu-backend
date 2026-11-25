package adminuser

import "time"

// UserListItem is one row in the admin user list. The password is never
// included — not even its hash — so the list can be shown or logged safely.
//
// AccountType tells the dashboard how the account signs in: "password" for a
// normal account, "oauth" for one that came in through Google and therefore
// has no password to reset.
type UserListItem struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar,omitempty"`
	Role     string `json:"role"`

	// AccountType is "password" or "oauth".
	AccountType string `json:"account_type"`

	EmailVerified bool `json:"email_verified"`

	// The institution a representative is bound to. Only one of these two is
	// ever set, and only for the "representative" role.
	RepresentativeUniversityID string `json:"representative_university_id,omitempty"`
	RepresentativeCollegeID    string `json:"representative_college_id,omitempty"`

	// The name of that institution, so the dashboard can show which one it
	// is instead of a bare UUID. Absent when the account is not a
	// representative, and also absent when the institution it was bound to
	// has since been deleted.
	RepresentativeUniversityName string `json:"representative_university_name,omitempty"`
	RepresentativeCollegeName    string `json:"representative_college_name,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserDetailResponse mirrors UserListItem. The admin endpoints always return
// the full row so the dashboard does not need a separate GET.
type UserDetailResponse struct {
	UserListItem
}

// UpdateRoleRequest is the body for PATCH /api/v1/admin/users/{id}/role.
//
// The scope is required only for the "representative" role, which must be
// bound to exactly one university or one college. It is ignored for the other
// two roles, and the existing scope is cleared when the role changes away
// from "representative".
type UpdateRoleRequest struct {
	Role string `json:"role" validate:"required" example:"representative"`

	// RepresentativeUniversityID binds the account to one university.
	RepresentativeUniversityID string `json:"representative_university_id,omitempty" example:"9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10"`

	// RepresentativeCollegeID binds the account to one college.
	RepresentativeCollegeID string `json:"representative_college_id,omitempty" example:"9a4f1b32-8e57-4c2f-bd63-2e7a8c4f9b10"`
}

// ResetPasswordResponse is what POST /api/v1/admin/users/{id}/reset-password
// returns.
//
// The admin never sees or sets the new password: a reset link is emailed to
// the account holder and they choose the password themselves. Sent tells the
// dashboard whether the email actually went out.
type ResetPasswordResponse struct {
	// Sent is true when the reset email was handed to the mailer.
	Sent bool `json:"sent"`

	// Email is the address the link was sent to.
	Email string `json:"email"`
}
