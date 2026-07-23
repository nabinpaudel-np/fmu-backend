package college

type CreateCollegeRequest struct {
	Name         string `json:"name"          validate:"required,max=255"`
	Slug         string `json:"slug"          validate:"required,min=2,max=255"`
	UniversityID string `json:"university_id" validate:"required,uuid"`
	Overview     string `json:"overview"      validate:"required"`

	Country      string `json:"country"       validate:"omitempty,max=100"`
	State        string `json:"state"         validate:"omitempty,max=100"`
	City         string `json:"city"          validate:"omitempty,max=100"`
	FullLocation string `json:"full_location" validate:"omitempty,max=255"`
	Logo         string `json:"logo"          validate:"omitempty,url"`
}

type CreateCollegeResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	UniversityID string `json:"university_id"`
	Overview     string `json:"overview"`
	Country      string `json:"country"`
	State        string `json:"state"`
	City         string `json:"city"`
	FullLocation string `json:"full_location"`
	Logo         string `json:"logo"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// UpdateCollegeRequest is the body for PATCH /colleges/{id}. All fields are
// optional; omitted fields are left unchanged. Pointer types let the caller
// distinguish "not provided" from "set to null" — the latter is only valid
// for the nullable columns (country, state, city, full_location, logo). The
// required columns (name, slug, overview) reject empty strings via validate.
type UpdateCollegeRequest struct {
	Name         *string `json:"name"          validate:"omitempty,min=1,max=255"`
	Slug         *string `json:"slug"          validate:"omitempty,min=2,max=255"`
	Overview     *string `json:"overview"      validate:"omitempty,min=1"`
	Country      *string `json:"country"       validate:"omitempty,max=100"`
	State        *string `json:"state"         validate:"omitempty,max=100"`
	City         *string `json:"city"          validate:"omitempty,max=100"`
	FullLocation *string `json:"full_location" validate:"omitempty,max=255"`
	Logo         *string `json:"logo"          validate:"omitempty,url"`
}

type CollegeListItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	UniversityID string `json:"university_id"`
	Country      string `json:"country"`
	State        string `json:"state"`
	City         string `json:"city"`
	Logo         string `json:"logo"`
	IsFavorited  bool   `json:"is_favorited"`
}

type CollegeDetailResponse struct {
	CreateCollegeResponse
}

// CollegeUniversitySummary is the parent-university projection embedded in
// search results so the client can render "College of Foo — Harvard" without
// a second request. Field shape matches CollegeListItem's string-only style.
type CollegeUniversitySummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Logo string `json:"logo"`
}

type CollegeSearchResult struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Slug        string                   `json:"slug"`
	University  CollegeUniversitySummary `json:"university"`
	Country     string                   `json:"country"`
	State       string                   `json:"state"`
	City        string                   `json:"city"`
	Logo        string                   `json:"logo"`
	IsFavorited bool                     `json:"is_favorited"`
}
