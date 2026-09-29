package scholarship

import "time"

// EducationLevelResponse / DemographicResponse / MajorResponse are the slim
// {id, name} shapes returned in detail payloads and the /lookups bundle.
type EducationLevelResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DemographicResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MajorResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateScholarshipRequest is the body for POST /scholarships. On a non-draft
// create every `required` field is enforced; `status:"draft"` bypasses the
// strict pass so an admin can stub the row and finish it later (only title +
// slug are required for a draft, enforced in the service).
type CreateScholarshipRequest struct {
	Title       string  `json:"title" validate:"required,max=255"`
	Slug        string  `json:"slug" validate:"required,max=255"`
	Description string  `json:"description" validate:"required"`
	AwardAmount string  `json:"award_amount" validate:"required,max=255"`
	AwardMin    float64 `json:"award_min" validate:"omitempty,gte=0"`
	AwardMax    float64 `json:"award_max" validate:"omitempty,gte=0"`
	Logo        string  `json:"logo" validate:"omitempty,url"`

	NumberOfAwards int32 `json:"number_of_awards" validate:"omitempty,gte=0"`
	IsRenewable    bool  `json:"is_renewable"`

	MinGpa                float64 `json:"min_gpa" validate:"omitempty,gte=0,lte=5"`
	RequiresFinancialNeed bool    `json:"requires_financial_need"`
	Country               string  `json:"country" validate:"omitempty,max=100"`
	State                 string  `json:"state" validate:"omitempty,max=100"`
	City                  string  `json:"city" validate:"omitempty,max=100"`

	ApplicationOpenDate   *time.Time `json:"application_open_date" validate:"omitempty"`
	ApplicationDeadline   *time.Time `json:"application_deadline" validate:"omitempty"`
	AwardNotificationDate *time.Time `json:"award_notification_date" validate:"omitempty"`

	EssayRequired                 bool   `json:"essay_required"`
	EssayPrompt                   string `json:"essay_prompt"`
	RecommendationLettersRequired int32  `json:"recommendation_letters_required" validate:"omitempty,gte=0"`
	TranscriptRequirement         string `json:"transcript_requirement" validate:"omitempty,oneof=none official unofficial"`
	PortfolioRequired             bool   `json:"portfolio_required"`
	ApplicationUrl                string `json:"application_url" validate:"omitempty,url,max=500"`

	ProviderType  string `json:"provider_type" validate:"omitempty,max=50"`
	ProviderName  string `json:"provider_name" validate:"omitempty,max=255"`
	ContactEmail  string `json:"contact_email" validate:"omitempty,email,max=255"`
	ContactPhone  string `json:"contact_phone" validate:"omitempty,max=50"`
	InternalNotes string `json:"internal_notes"`
	UniversityID  string `json:"university_id" validate:"omitempty,uuid"`
	CollegeID     string `json:"college_id" validate:"omitempty,uuid"`

	SeoTitle       string `json:"seo_title" validate:"omitempty,max=70"`
	SeoDescription string `json:"seo_description" validate:"omitempty,max=160"`
	IsPopular      bool   `json:"is_popular"`
	IsFeatured     bool   `json:"is_featured"`

	EducationLevelIDs []string `json:"education_level_ids" validate:"omitempty,dive,uuid"`
	MajorIDs          []string `json:"major_ids" validate:"omitempty,dive,uuid"`
	DemographicIDs    []string `json:"demographic_ids" validate:"omitempty,dive,uuid"`

	Status string `json:"status" validate:"omitempty,oneof=draft published"`
}

// UpdateScholarshipRequest is the body for PATCH /scholarships/{id}. All fields
// are optional — pointer types distinguish "omit" (no change) from "send the
// zero value" (explicit update). The `*[]string` association fields are nil =
// leave untouched, non-nil (including []) = replace the entire list.
type UpdateScholarshipRequest struct {
	Title       *string  `json:"title,omitempty" validate:"omitempty,max=255"`
	Slug        *string  `json:"slug,omitempty" validate:"omitempty,min=2,max=255"`
	Description *string  `json:"description,omitempty"`
	AwardAmount *string  `json:"award_amount,omitempty" validate:"omitempty,max=255"`
	AwardMin    *float64 `json:"award_min,omitempty" validate:"omitempty,gte=0"`
	AwardMax    *float64 `json:"award_max,omitempty" validate:"omitempty,gte=0"`
	Logo        *string  `json:"logo,omitempty" validate:"omitempty,url"`

	NumberOfAwards *int32 `json:"number_of_awards,omitempty" validate:"omitempty,gte=0"`
	IsRenewable    *bool  `json:"is_renewable,omitempty"`

	MinGpa                *float64 `json:"min_gpa,omitempty" validate:"omitempty,gte=0,lte=5"`
	RequiresFinancialNeed *bool    `json:"requires_financial_need,omitempty"`
	Country               *string  `json:"country,omitempty" validate:"omitempty,max=100"`
	State                 *string  `json:"state,omitempty" validate:"omitempty,max=100"`
	City                  *string  `json:"city,omitempty" validate:"omitempty,max=100"`

	ApplicationOpenDate   *time.Time `json:"application_open_date,omitempty"`
	ApplicationDeadline   *time.Time `json:"application_deadline,omitempty"`
	AwardNotificationDate *time.Time `json:"award_notification_date,omitempty"`

	EssayRequired                 *bool   `json:"essay_required,omitempty"`
	EssayPrompt                   *string `json:"essay_prompt,omitempty"`
	RecommendationLettersRequired *int32  `json:"recommendation_letters_required,omitempty" validate:"omitempty,gte=0"`
	TranscriptRequirement         *string `json:"transcript_requirement,omitempty" validate:"omitempty,oneof=none official unofficial"`
	PortfolioRequired             *bool   `json:"portfolio_required,omitempty"`
	ApplicationUrl                *string `json:"application_url,omitempty" validate:"omitempty,url,max=500"`

	ProviderType  *string `json:"provider_type,omitempty" validate:"omitempty,max=50"`
	ProviderName  *string `json:"provider_name,omitempty" validate:"omitempty,max=255"`
	ContactEmail  *string `json:"contact_email,omitempty" validate:"omitempty,email,max=255"`
	ContactPhone  *string `json:"contact_phone,omitempty" validate:"omitempty,max=50"`
	InternalNotes *string `json:"internal_notes,omitempty"`
	UniversityID  *string `json:"university_id,omitempty" validate:"omitempty,uuid"`
	CollegeID     *string `json:"college_id,omitempty" validate:"omitempty,uuid"`

	SeoTitle       *string `json:"seo_title,omitempty" validate:"omitempty,max=70"`
	SeoDescription *string `json:"seo_description,omitempty" validate:"omitempty,max=160"`
	IsPopular      *bool   `json:"is_popular,omitempty"`
	IsFeatured     *bool   `json:"is_featured,omitempty"`

	EducationLevelIDs *[]string `json:"education_level_ids,omitempty" validate:"omitempty,dive,uuid"`
	MajorIDs          *[]string `json:"major_ids,omitempty" validate:"omitempty,dive,uuid"`
	DemographicIDs    *[]string `json:"demographic_ids,omitempty" validate:"omitempty,dive,uuid"`
}

// ScholarshipResponse is the full row shape returned by the admin-only
// create/update/publish endpoints. InternalNotes is admin-only — the public
// GET handlers strip it before responding.
type ScholarshipResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	AwardAmount string  `json:"award_amount"`
	AwardMin    float64 `json:"award_min"`
	AwardMax    float64 `json:"award_max"`
	Logo        string  `json:"logo"`

	NumberOfAwards int32 `json:"number_of_awards"`
	IsRenewable    bool  `json:"is_renewable"`

	MinGpa                float64 `json:"min_gpa"`
	RequiresFinancialNeed bool    `json:"requires_financial_need"`
	Country               string  `json:"country"`
	State                 string  `json:"state"`
	City                  string  `json:"city"`

	ApplicationOpenDate   *time.Time `json:"application_open_date"`
	ApplicationDeadline   *time.Time `json:"application_deadline"`
	AwardNotificationDate *time.Time `json:"award_notification_date"`

	EssayRequired                 bool   `json:"essay_required"`
	EssayPrompt                   string `json:"essay_prompt"`
	RecommendationLettersRequired int32  `json:"recommendation_letters_required"`
	TranscriptRequirement         string `json:"transcript_requirement"`
	PortfolioRequired             bool   `json:"portfolio_required"`
	ApplicationUrl                string `json:"application_url"`

	ProviderType string `json:"provider_type"`
	ProviderName string `json:"provider_name"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	// InternalNotes is admin-only; omitted from public responses.
	InternalNotes *string `json:"internal_notes,omitempty"`
	UniversityID  *string `json:"university_id"`
	CollegeID     *string `json:"college_id"`

	SeoTitle       string `json:"seo_title"`
	SeoDescription string `json:"seo_description"`
	IsPopular      bool   `json:"is_popular"`
	IsFeatured     bool   `json:"is_featured"`

	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ScholarshipListItem is the slim shape returned by the paginated list and the
// favorites list. It carries the fields a search card needs plus the facet
// flags used for client-side badges.
type ScholarshipListItem struct {
	ID                    string     `json:"id"`
	Title                 string     `json:"title"`
	Slug                  string     `json:"slug"`
	AwardAmount           string     `json:"award_amount"`
	Logo                  string     `json:"logo"`
	ProviderType          string     `json:"provider_type"`
	ProviderName          string     `json:"provider_name"`
	Country               string     `json:"country"`
	ApplicationDeadline   *time.Time `json:"application_deadline"`
	MinGpa                float64    `json:"min_gpa"`
	RequiresFinancialNeed bool       `json:"requires_financial_need"`
	EssayRequired         bool       `json:"essay_required"`
	IsRenewable           bool       `json:"is_renewable"`
	IsPopular             bool       `json:"is_popular"`
	IsFeatured            bool       `json:"is_featured"`
	IsFavorited           bool       `json:"is_favorited"`
}

// ScholarshipDetailResponse is the full detail payload for GET /{id} — the row
// plus its resolved multi-value associations and the per-user is_favorited flag.
type ScholarshipDetailResponse struct {
	ScholarshipResponse
	EducationLevels []EducationLevelResponse `json:"education_levels"`
	Majors          []MajorResponse          `json:"majors"`
	Demographics    []DemographicResponse    `json:"demographics"`
	IsFavorited     bool                     `json:"is_favorited"`
}

// ScholarshipSearchResult is the slim typeahead shape for /scholarships/search.
type ScholarshipSearchResult struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	ProviderName string `json:"provider_name"`
	ProviderType string `json:"provider_type"`
	Logo         string `json:"logo"`
	AwardAmount  string `json:"award_amount"`
	IsFavorited  bool   `json:"is_favorited"`
}

// AllScholarshipLookupsResponse is the bundled reference data for the create/
// filter forms. ProviderTypes is a static list (not a lookup table).
type AllScholarshipLookupsResponse struct {
	EducationLevels []EducationLevelResponse `json:"education_levels"`
	Demographics    []DemographicResponse    `json:"demographics"`
	Majors          []MajorResponse          `json:"majors"`
	ProviderTypes   []string                 `json:"provider_types"`
}
