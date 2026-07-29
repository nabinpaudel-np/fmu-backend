package college

import "time"

type CreateCollegeRequest struct {
	Name         string `json:"name"          validate:"required,max=255"`
	Slug         string `json:"slug"          validate:"required,min=2,max=255"`
	UniversityID string `json:"university_id" validate:"required,uuid"`
	Overview     string `json:"overview"      validate:"required"`
	Excerpt      string `json:"excerpt"       validate:"omitempty,max=500"`

	Country      string `json:"country"       validate:"omitempty,max=100"`
	State        string `json:"state"         validate:"omitempty,max=100"`
	City         string `json:"city"          validate:"omitempty,max=100"`
	FullLocation string `json:"full_location" validate:"omitempty,max=255"`
	Zipcode      string `json:"zipcode"       validate:"omitempty,max=20"`
	FullAddress  string `json:"full_address"  validate:"omitempty,max=500"`

	CoverImage string `json:"cover_image" validate:"omitempty,url"`
	Logo       string `json:"logo"        validate:"omitempty,url"`
	MapsUrl    string `json:"maps_url"    validate:"omitempty,url,max=500"`
	GalleryImages []string `json:"gallery_images" validate:"omitempty,dive,url"`

	InstitutionType string `json:"institution_type" validate:"omitempty,max=50"`
	CampusSetting   string `json:"campus_setting"   validate:"omitempty,max=50"`

	ContactEmail string `json:"contact_email" validate:"omitempty,email,max=255"`
	ContactPhone string `json:"contact_phone" validate:"omitempty,max=50"`
	Website      string `json:"website"       validate:"omitempty,url,max=500"`

	FoundedYear int32  `json:"founded_year" validate:"omitempty,gte=1000,lte=2100"`
	CampusSize  string `json:"campus_size"  validate:"omitempty,max=100"`
	IsPopular   bool   `json:"is_popular"`
	IsFeatured  bool   `json:"is_featured"`

	SeoTitle       string `json:"seo_title"       validate:"omitempty,max=70"`
	SeoDescription string `json:"seo_description" validate:"omitempty,max=160"`

	// Lookup associations — all optional on both published and draft rows.
	// Each item must be a UUID of an existing row in the referenced lookup table.
	// Unknown IDs return 400 with the field list (see InvalidReferencesError).
	DegreeLevelIDs []string `json:"degree_level_ids" validate:"omitempty,dive,uuid"`
	MajorIDs       []string `json:"major_ids"       validate:"omitempty,dive,uuid"`
	StudyFormatIDs []string `json:"study_format_ids" validate:"omitempty,dive,uuid"`

	// Status selects the lifecycle stage on create. Defaults to "published"
	// (every field is then required). "draft" loosens validation to just
	// `name` + `slug` so the row can be saved and finished later.
	Status string `json:"status" validate:"omitempty,oneof=draft published" example:"published"`
}

type CreateCollegeResponse struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Slug            string   `json:"slug"`
	UniversityID    string   `json:"university_id"`
	Overview        string   `json:"overview"`
	Excerpt         string   `json:"excerpt"`
	Country         string   `json:"country"`
	State           string   `json:"state"`
	City            string   `json:"city"`
	FullLocation    string   `json:"full_location"`
	Zipcode         string   `json:"zipcode"`
	FullAddress     string   `json:"full_address"`
	CoverImage      string   `json:"cover_image"`
	Logo            string   `json:"logo"`
	MapsUrl         string   `json:"maps_url"`
	GalleryImages   []string `json:"gallery_images"`
	InstitutionType string   `json:"institution_type"`
	CampusSetting   string   `json:"campus_setting"`
	ContactEmail    string   `json:"contact_email"`
	ContactPhone    string   `json:"contact_phone"`
	Website         string   `json:"website"`
	FoundedYear     int32    `json:"founded_year"`
	CampusSize      string   `json:"campus_size"`
	IsPopular       bool     `json:"is_popular"`
	IsFeatured      bool     `json:"is_featured"`
	SeoTitle        string   `json:"seo_title"`
	SeoDescription  string   `json:"seo_description"`
	Status          string   `json:"status" example:"published"`
	PublishedAt     *time.Time `json:"published_at" swaggertype:"string" example:"2026-07-30T12:00:00Z"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type UpdateCollegeRequest struct {
	Name         *string `json:"name"          validate:"omitempty,min=1,max=255"`
	Slug         *string `json:"slug"          validate:"omitempty,min=2,max=255"`
	Overview     *string `json:"overview"      validate:"omitempty,min=1"`
	Excerpt      *string `json:"excerpt"       validate:"omitempty,max=500"`
	Country      *string `json:"country"       validate:"omitempty,max=100"`
	State        *string `json:"state"         validate:"omitempty,max=100"`
	City         *string `json:"city"          validate:"omitempty,max=100"`
	FullLocation *string `json:"full_location" validate:"omitempty,max=255"`
	Zipcode      *string `json:"zipcode"       validate:"omitempty,max=20"`
	FullAddress  *string `json:"full_address"  validate:"omitempty,max=500"`

	CoverImage *string   `json:"cover_image"    validate:"omitempty,url"`
	Logo       *string   `json:"logo"           validate:"omitempty,url"`
	MapsUrl    *string   `json:"maps_url"       validate:"omitempty,url,max=500"`
	GalleryImages *[]string `json:"gallery_images" validate:"omitempty,dive,url"`

	InstitutionType *string `json:"institution_type" validate:"omitempty,max=50"`
	CampusSetting   *string `json:"campus_setting"   validate:"omitempty,max=50"`

	ContactEmail *string `json:"contact_email" validate:"omitempty,email,max=255"`
	ContactPhone *string `json:"contact_phone" validate:"omitempty,max=50"`
	Website      *string `json:"website"       validate:"omitempty,url,max=500"`

	FoundedYear *int32  `json:"founded_year" validate:"omitempty,gte=1000,lte=2100"`
	CampusSize  *string `json:"campus_size"  validate:"omitempty,max=100"`
	IsPopular   *bool   `json:"is_popular"`
	IsFeatured  *bool   `json:"is_featured"`

	SeoTitle       *string `json:"seo_title"       validate:"omitempty,max=70"`
	SeoDescription *string `json:"seo_description" validate:"omitempty,max=160"`

	// Lookup associations: nil = leave existing rows untouched, non-nil
	// (including an empty slice) replaces the entire association list.
	DegreeLevelIDs *[]string `json:"degree_level_ids"  validate:"omitempty,dive,uuid"`
	MajorIDs       *[]string `json:"major_ids"        validate:"omitempty,dive,uuid"`
	StudyFormatIDs *[]string `json:"study_format_ids" validate:"omitempty,dive,uuid"`
}

type CollegeListItem struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	UniversityID      string `json:"university_id"`
	Country           string `json:"country"`
	State             string `json:"state"`
	City              string `json:"city"`
	Logo              string `json:"logo"`
	CoverImage        string `json:"cover_image"`
	InstitutionType   string `json:"institution_type"`
	CampusSetting     string `json:"campus_setting"`
	IsPopular         bool   `json:"is_popular"`
	IsFeatured        bool   `json:"is_featured"`
	IsFavorited       bool   `json:"is_favorited"`
	HasRepresentative bool   `json:"has_representative"`
}

type CollegeDetailResponse struct {
	CreateCollegeResponse
	DegreeLevels       []CollegeDegreeLevelResponse `json:"degree_levels"`
	Majors             []CollegeMajorResponse       `json:"majors"`
	StudyFormats       []CollegeStudyFormatResponse `json:"study_formats"`
	HasRepresentative  bool                         `json:"has_representative"`
}

// Lookup response types — mirror the shape of the rows in the corresponding
// lookup tables. Returned only on the detail endpoint; the list/search rows
// stay slim. Kept in this package to avoid an inter-package import from
// internal/university for types that conceptually belong to the lookup
// tables but are surfaced through the college DTO.
type CollegeDegreeLevelResponse struct {
	ID   string `json:"id"   example:"5b7e1c91-006a-407b-a9bd-609f60cefa0a"`
	Name string `json:"name" example:"Bachelor's"`
}

type CollegeMajorResponse struct {
	ID   string `json:"id"   example:"125479fb-fccb-43cc-980a-84e1d73117b3"`
	Name string `json:"name" example:"Computer Science"`
}

type CollegeStudyFormatResponse struct {
	ID   string `json:"id"   example:"55896b33-58bd-44cd-bf75-6387dd5614d4"`
	Name string `json:"name" example:"On-Campus"`
}

type CollegeUniversitySummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Logo string `json:"logo"`
}

type CollegeSearchResult struct {
	ID                string                   `json:"id"`
	Name              string                   `json:"name"`
	Slug              string                   `json:"slug"`
	University        CollegeUniversitySummary `json:"university"`
	Country           string                   `json:"country"`
	State             string                   `json:"state"`
	City              string                   `json:"city"`
	Logo              string                   `json:"logo"`
	IsFavorited       bool                     `json:"is_favorited"`
	HasRepresentative bool                     `json:"has_representative"`
}
