package counselling

import "time"

// TargetType discriminates which entity a counselling inquiry is attached
// to. TargetGeneral is the "not tied to any institution" form; the URL
// prefix and the target_id nullable field on the row both derive from this.
type TargetType string

const (
	TargetGeneral    TargetType = "general"
	TargetUniversity TargetType = "university"
	TargetCollege    TargetType = "college"
)

// IsValid reports whether t is a known target type. Used by handlers to
// reject bad ?type= filter values on the list endpoint.
func (t TargetType) IsValid() bool {
	switch t {
	case TargetGeneral, TargetUniversity, TargetCollege:
		return true
	}
	return false
}

const (
	StatusPending  = "pending"
	StatusReviewed = "reviewed"
	StatusArchived = "archived"
)

// SubmitGeneralRequest is the body for POST /counselling/general. Required
// fields mirror the frontend "Free Counselling" form; resume_url is optional.
type SubmitGeneralRequest struct {
	FullName            string `json:"full_name"            validate:"required,min=2,max=255"`
	Email               string `json:"email"                validate:"required,email,max=255"`
	Phone               string `json:"phone"                validate:"omitempty,max=50"`
	Country             string `json:"country"              validate:"required,min=2,max=100"`
	PreferredUniversity string `json:"preferred_university" validate:"omitempty,max=255"`
	ResumeURL           string `json:"resume_url"           validate:"omitempty,url,max=500"`
}

// SubmitSpecificRequest is the body for POST /counselling/universities/{id}
// and /counselling/colleges/{id}. The institution id comes from the URL, not
// the body. Phone and start_term are required per the frontend form.
type SubmitSpecificRequest struct {
	FullName          string `json:"full_name"          validate:"required,min=2,max=255"`
	Email             string `json:"email"              validate:"required,email,max=255"`
	Phone             string `json:"phone"              validate:"required,min=4,max=50"`
	Country           string `json:"country"            validate:"omitempty,max=100"`
	ProgramOfInterest string `json:"program_of_interest" validate:"omitempty,max=255"`
	StartTerm         string `json:"start_term"         validate:"required,min=1,max=100"`
	CurrentEducation  string `json:"current_education"  validate:"omitempty,max=100"`
	TestScores        string `json:"test_scores"        validate:"omitempty,max=1000"`
	Message           string `json:"message"            validate:"omitempty,max=5000"`
}

// SubmitResponse is what the three public submit endpoints return. We expose
// only the inquiry id + status; admin/rep dashboards see the rest.
type SubmitResponse struct {
	InquiryID string     `json:"inquiry_id"`
	Type      TargetType `json:"type"`
	TargetID  *string    `json:"target_id,omitempty"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

// CounsellingListItem is the row shape returned by GET admin/representative
// list endpoints. Fields are flattened into one struct since the table holds
// all variants; nullable DB columns map to omitempty JSON fields.
type CounsellingListItem struct {
	ID                  string     `json:"id"`
	Type                TargetType `json:"type"`
	TargetID            *string    `json:"target_id,omitempty"`
	TargetName          string     `json:"target_name,omitempty"`
	FullName            string     `json:"full_name"`
	Email               string     `json:"email"`
	Phone               string     `json:"phone,omitempty"`
	Country             string     `json:"country,omitempty"`
	PreferredUniversity string     `json:"preferred_university,omitempty"`
	ProgramOfInterest   string     `json:"program_of_interest,omitempty"`
	StartTerm           string     `json:"start_term,omitempty"`
	CurrentEducation    string     `json:"current_education,omitempty"`
	TestScores          string     `json:"test_scores,omitempty"`
	Message             string     `json:"message,omitempty"`
	ResumeURL           string     `json:"resume_url,omitempty"`
	Status              string     `json:"status"`
	ReviewerID          *string    `json:"reviewer_id,omitempty"`
	ReviewedAt          *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote          string     `json:"review_note,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// UpdateRequest is the body for PATCH .../counselling/{id}. Status transitions
// are limited to reviewed or archived (admin/rep cannot re-open a row to
// pending once it's been touched).
type UpdateRequest struct {
	Status     string `json:"status"      validate:"required,oneof=reviewed archived"`
	ReviewNote string `json:"review_note" validate:"omitempty,max=2000"`
}
