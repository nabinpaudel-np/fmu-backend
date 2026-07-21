package claim

import "time"

// SubmitClaimRequest is the public-facing payload for POST /claims/universities/{id}.
type SubmitClaimRequest struct {
	FullName    string `json:"full_name"    validate:"required,min=2,max=255"`
	WorkEmail   string `json:"work_email"   validate:"required,email,max=255"`
	DocumentURL string `json:"document_url" validate:"required,url,max=500"`
}

// SubmitClaimResponse is what the public submit endpoint returns. We expose only
// the claim id + status; the admin reviews the rest in the dashboard.
type SubmitClaimResponse struct {
	ClaimID      string    `json:"claim_id"`
	UniversityID string    `json:"university_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// ClaimListItem is the row shape used by GET /admin/claims.
type ClaimListItem struct {
	ID             string     `json:"id"`
	UniversityID   string     `json:"university_id"`
	UniversityName string     `json:"university_name"`
	FullName       string     `json:"full_name"`
	WorkEmail      string     `json:"work_email"`
	DocumentURL    string     `json:"document_url"`
	Status         string     `json:"status"`
	ReviewerID     *string    `json:"reviewer_id,omitempty"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote     *string    `json:"review_note,omitempty"`
	CreatedUserID  *string    `json:"created_user_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ClaimDetailResponse mirrors ClaimListItem; admin endpoints always return the
// full row so the dashboard doesn't need a separate GET.
type ClaimDetailResponse struct {
	ClaimListItem
}

// ReviewDecisionRequest is the body for both approve and reject. ReviewNote is
// optional but recommended on rejection so the claimant knows why.
type ReviewDecisionRequest struct {
	ReviewNote string `json:"review_note" validate:"omitempty,max=1000"`
}

// ApproveClaimResponse is what POST /admin/claims/{id}/approve returns.
// PlainPassword is returned exactly once — the admin must deliver it to the new
// representative manually. There is no other way to retrieve it afterwards.
type ApproveClaimResponse struct {
	Claim         ClaimListItem `json:"claim"`
	CreatedUserID string        `json:"created_user_id"`
	Email         string        `json:"email"`
	PlainPassword string        `json:"plain_password"`
	Role          string        `json:"role"`
}

// RejectClaimResponse is what POST /admin/claims/{id}/reject returns.
type RejectClaimResponse struct {
	Claim ClaimListItem `json:"claim"`
}
