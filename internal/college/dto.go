package college

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

	CoverImage    string   `json:"cover_image"    validate:"omitempty,url"`
	Logo          string   `json:"logo"           validate:"omitempty,url"`
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
	CoverImage      string   `json:"cover_image"`
	Logo            string   `json:"logo"`
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

	CoverImage    *string   `json:"cover_image"    validate:"omitempty,url"`
	Logo          *string   `json:"logo"           validate:"omitempty,url"`
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
	HasRepresentative bool `json:"has_representative"`
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
