package nlsearch

import (
	"time"

	"fmu-backend/internal/pagination"
)

// The response carries one item shape for all four entity types, so the
// frontend can render one result list. The `entity` field says what each row
// is and `title` holds its name.

// ResultItem is one row in a result list.
type ResultItem struct {
	Entity string `json:"entity"`
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`

	Logo         string `json:"logo,omitempty"`
	CoverImage   string `json:"cover_image,omitempty"`
	Country      string `json:"country,omitempty"`
	State        string `json:"state,omitempty"`
	City         string `json:"city,omitempty"`
	FullLocation string `json:"full_location,omitempty"`

	InstitutionType string `json:"institution_type,omitempty"`
	CampusSetting   string `json:"campus_setting,omitempty"`

	TuitionMin     float64 `json:"tuition_min,omitempty"`
	TuitionMax     float64 `json:"tuition_max,omitempty"`
	AcceptanceRate float64 `json:"acceptance_rate,omitempty"`
	EmploymentRate float64 `json:"employment_rate,omitempty"`
	FoundedYear    int     `json:"founded_year,omitempty"`

	AwardAmount    string  `json:"award_amount,omitempty"`
	AwardMin       float64 `json:"award_min,omitempty"`
	AwardMax       float64 `json:"award_max,omitempty"`
	NumberOfAwards int     `json:"number_of_awards,omitempty"`
	MinGpa         float64 `json:"min_gpa,omitempty"`

	ProviderName string `json:"provider_name,omitempty"`
	ProviderType string `json:"provider_type,omitempty"`

	Description   string `json:"description,omitempty"`
	Excerpt       string `json:"excerpt,omitempty"`
	CareerOptions string `json:"career_options,omitempty"`

	ApplicationDeadline *time.Time `json:"application_deadline,omitempty"`

	IsPopular  bool `json:"is_popular"`
	IsFeatured bool `json:"is_featured"`
}

// Interpretation tells the caller how the question was read. It is shown to
// the user; nothing in it affects the query.
type Interpretation struct {
	Entity      string   `json:"entity"`
	Assumptions string   `json:"assumptions,omitempty"`
	Unmapped    []string `json:"unmapped,omitempty"`
}

// SearchRequest is the request body.
type SearchRequest struct {
	Query    string `json:"query" example:"universities in kathmandu that offer scholarships"`
	Page     int    `json:"page,omitempty" example:"1"`
	PageSize int    `json:"page_size,omitempty" example:"20"`
}

// SearchResponse is the payload of a successful search.
type SearchResponse struct {
	Query          string          `json:"query"`
	Interpretation Interpretation  `json:"interpretation"`
	Items          []ResultItem    `json:"items"`
	Meta           pagination.Meta `json:"meta"`
}
