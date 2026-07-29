package university

import (
	"time"

	"fmu-backend/internal/programs"
)

type MajorResponse struct {
	ID   string `json:"id" example:"125479fb-fccb-43cc-980a-84e1d73117b3"`
	Name string `json:"name" example:"Computer Science"`
}

type DegreeLevelResponse struct {
	ID   string `json:"id" example:"5b7e1c91-006a-407b-a9bd-609f60cefa0a"`
	Name string `json:"name" example:"Bachelor's"`
}

type StudyFormatResponse struct {
	ID   string `json:"id" example:"55896b33-58bd-44cd-bf75-6387dd5614d4"`
	Name string `json:"name" example:"On-Campus"`
}

type SpecialAffiliationResponse struct {
	ID   string `json:"id" example:"8d727958-85ee-4ef3-bb63-a472d7541c59"`
	Name string `json:"name" example:"HBCU"`
}

type AthleticResponse struct {
	ID   string `json:"id" example:"fa19a9f6-d650-4d85-a873-514f197b07b5"`
	Name string `json:"name" example:"NCAA Division I"`
}

type SupportServiceResponse struct {
	ID   string `json:"id" example:"f04ab66d-5cab-40fe-a372-6d68a786e60b"`
	Name string `json:"name" example:"Tutoring"`
}

type CreateUniversityRequest struct {
	Name     string `json:"name" validate:"required" example:"Massachusetts Institute of Technology"`
	Slug     string `json:"slug" validate:"required" example:"mit"`
	Overview string `json:"overview" validate:"required" example:"MIT is a private research university in Cambridge, Massachusetts."`
	Excerpt  string `json:"excerpt" validate:"omitempty,max=500" example:"World-class research university founded in 1861."`

	Country      string `json:"country" validate:"required" example:"US"`
	State        string `json:"state" validate:"omitempty" example:"MA"`
	City         string `json:"city" validate:"required" example:"Cambridge"`
	FullLocation string `json:"full_location" validate:"omitempty" example:"Cambridge, MA, US"`
	Zipcode      string `json:"zipcode" validate:"omitempty" example:"02139"`

	CoverImage string `json:"cover_image" validate:"omitempty,url" example:"https://cdn.example.com/mit-cover.jpg"`
	Logo       string `json:"logo" validate:"omitempty,url" example:"https://cdn.example.com/mit-logo.png"`

	InstitutionType string `json:"institution_type" validate:"required" example:"Private"`
	CampusSetting   string `json:"campus_setting" validate:"required" example:"Urban"`

	InStateTuition       float64 `json:"in_state_tuition" validate:"omitempty,gte=0" example:"57590"`
	OutOfStateTuition    float64 `json:"out_of_state_tuition" validate:"omitempty,gte=0" example:"57590"`
	InternationalTuition float64 `json:"international_tuition" validate:"omitempty,gte=0" example:"57590"`
	TuitionMin           int32   `json:"tuition_min" validate:"omitempty,gte=0" example:"10000"`
	TuitionMax           int32   `json:"tuition_max" validate:"omitempty,gte=0" example:"25000"`

	NeedBasedAid      bool `json:"need_based_aid" example:"true"`
	MeritScholarships bool `json:"merit_scholarships" example:"true"`
	WorkStudy         bool `json:"work_study" example:"true"`
	NoApplicationFee  bool `json:"no_application_fee" example:"false"`

	AcceptanceRate float64 `json:"acceptance_rate" validate:"omitempty,gte=0,lte=100" example:"4.3"`
	TestingPolicy  string  `json:"testing_policy" validate:"omitempty" example:"Optional"`
	SatRange       string  `json:"sat_range" validate:"omitempty" example:"1500-1570"`
	ActRange       string  `json:"act_range" validate:"omitempty" example:"34-36"`

	OnCampusHousing          bool `json:"on_campus_housing" example:"true"`
	FreshmenRequiredOnCampus bool `json:"freshmen_required_on_campus" example:"true"`

	ContactEmail string `json:"contact_email" validate:"required,email" example:"admissions@mit.edu"`
	ContactPhone string `json:"contact_phone" validate:"omitempty" example:"+1-617-253-1000"`
	Website      string `json:"website" validate:"required,url" example:"https://www.mit.edu"`

	AvgHighSchoolGpa float64  `json:"avg_high_school_gpa" validate:"omitempty,gte=0,lte=5" example:"3.8"`
	FoundedYear      int32    `json:"founded_year" validate:"omitempty,gte=1000,lte=2100" example:"1861"`
	CampusSize       string   `json:"campus_size" validate:"omitempty" example:"168 acres"`
	GalleryImages    []string `json:"gallery_images" validate:"omitempty,dive,url" example:"https://cdn.example.com/mit-1.jpg,https://cdn.example.com/mit-2.jpg"`
	IsPopular        bool     `json:"is_popular" example:"true"`
	IsFeatured       bool     `json:"is_featured" example:"true"`

	MapsUrl          string  `json:"maps_url" validate:"omitempty,url,max=500" example:"https://maps.google.com/?q=MIT"`
	FullAddress      string  `json:"full_address" validate:"omitempty,max=500" example:"77 Massachusetts Ave, Cambridge, MA 02139"`
	EmploymentRate   float64 `json:"employment_rate" validate:"omitempty,gte=0,lte=100" example:"92.5"`
	ResearchOutput   string  `json:"research_output" validate:"omitempty,max=50" example:"R1"`
	HousingType      string  `json:"housing_type" validate:"omitempty,max=50" example:"On-Campus"`
	SeoTitle         string  `json:"seo_title" validate:"omitempty,max=70" example:"MIT - Top Research University"`
	SeoDescription   string  `json:"seo_description" validate:"omitempty,max=160" example:"Learn about MIT admissions, programs, and campus life."`

	DegreeLevelIDs        []string `json:"degree_level_ids" validate:"required,min=1,dive,uuid"`
	MajorIDs              []string `json:"major_ids" validate:"required,min=1,dive,uuid"`
	StudyFormatIDs        []string `json:"study_format_ids" validate:"omitempty,dive,uuid"`
	SpecialAffiliationIDs []string `json:"special_affiliation_ids" validate:"omitempty,dive,uuid"`
	AthleticIDs           []string `json:"athletic_ids" validate:"omitempty,dive,uuid"`
	SupportServiceIDs     []string `json:"support_service_ids" validate:"omitempty,dive,uuid"`

	// Status selects the lifecycle stage on create. Defaults to "published"
	// (every field is then required). "draft" loosens validation to just
	// `name` + `slug` so an admin can stub the row and finish it later.
	Status string `json:"status" validate:"omitempty,oneof=draft published" example:"published"`
}

// PatchUniversityRequest is the body for PATCH /universities/{id}. All fields
// are optional — pointer types let us distinguish "omit" (no change) from
// "send the zero value" (explicit update).
type PatchUniversityRequest struct {
	Name     *string `json:"name,omitempty"      validate:"omitempty,max=255"`
	Slug     *string `json:"slug,omitempty"      validate:"omitempty,min=2,max=255"`
	Overview *string `json:"overview,omitempty"`
	Excerpt  *string `json:"excerpt,omitempty"   validate:"omitempty,max=500"`

	Country      *string `json:"country,omitempty"       validate:"omitempty,max=100"`
	State        *string `json:"state,omitempty"         validate:"omitempty,max=100"`
	City         *string `json:"city,omitempty"          validate:"omitempty,max=100"`
	FullLocation *string `json:"full_location,omitempty" validate:"omitempty,max=255"`
	Zipcode      *string `json:"zipcode,omitempty"`

	CoverImage *string `json:"cover_image,omitempty" validate:"omitempty,url"`
	Logo       *string `json:"logo,omitempty"       validate:"omitempty,url"`

	InstitutionType *string `json:"institution_type,omitempty" validate:"omitempty,max=50"`
	CampusSetting   *string `json:"campus_setting,omitempty"   validate:"omitempty,max=50"`

	InStateTuition       *float64 `json:"in_state_tuition,omitempty"       validate:"omitempty,gte=0"`
	OutOfStateTuition    *float64 `json:"out_of_state_tuition,omitempty"    validate:"omitempty,gte=0"`
	InternationalTuition *float64 `json:"international_tuition,omitempty" validate:"omitempty,gte=0"`
	TuitionMin           *int32   `json:"tuition_min,omitempty"            validate:"omitempty,gte=0"`
	TuitionMax           *int32   `json:"tuition_max,omitempty"            validate:"omitempty,gte=0"`

	NeedBasedAid      *bool `json:"need_based_aid,omitempty"`
	MeritScholarships *bool `json:"merit_scholarships,omitempty"`
	WorkStudy         *bool `json:"work_study,omitempty"`
	NoApplicationFee  *bool `json:"no_application_fee,omitempty"`

	AcceptanceRate *float64 `json:"acceptance_rate,omitempty" validate:"omitempty,gte=0,lte=100"`
	TestingPolicy  *string  `json:"testing_policy,omitempty"  validate:"omitempty,max=50"`
	SatRange       *string  `json:"sat_range,omitempty"       validate:"omitempty,max=20"`
	ActRange       *string  `json:"act_range,omitempty"       validate:"omitempty,max=20"`

	OnCampusHousing          *bool `json:"on_campus_housing,omitempty"`
	FreshmenRequiredOnCampus *bool `json:"freshmen_required_on_campus,omitempty"`

	ContactEmail *string `json:"contact_email,omitempty" validate:"omitempty,email,max=255"`
	ContactPhone *string `json:"contact_phone,omitempty" validate:"omitempty,max=50"`
	Website      *string `json:"website,omitempty"       validate:"omitempty,url,max=500"`

	AvgHighSchoolGpa *float64  `json:"avg_high_school_gpa,omitempty" validate:"omitempty,gte=0,lte=5"`
	FoundedYear      *int32    `json:"founded_year,omitempty"        validate:"omitempty,gte=1000,lte=2100"`
	CampusSize       *string   `json:"campus_size,omitempty"         validate:"omitempty,max=100"`
	GalleryImages    *[]string `json:"gallery_images,omitempty"      validate:"omitempty,dive,url"`
	IsPopular        *bool     `json:"is_popular,omitempty"`
	IsFeatured       *bool     `json:"is_featured,omitempty"`

	MapsUrl        *string  `json:"maps_url,omitempty"        validate:"omitempty,url,max=500"`
	FullAddress    *string  `json:"full_address,omitempty"    validate:"omitempty,max=500"`
	EmploymentRate *float64 `json:"employment_rate,omitempty" validate:"omitempty,gte=0,lte=100"`
	ResearchOutput *string  `json:"research_output,omitempty" validate:"omitempty,max=50"`
	HousingType    *string  `json:"housing_type,omitempty"    validate:"omitempty,max=50"`
	SeoTitle       *string  `json:"seo_title,omitempty"       validate:"omitempty,max=70"`
	SeoDescription *string  `json:"seo_description,omitempty" validate:"omitempty,max=160"`

	// nil = leave existing associations untouched; non-nil (including [])
	// = replace the entire association list.
	DegreeLevelIDs        *[]string `json:"degree_level_ids,omitempty"        validate:"omitempty,dive,uuid"`
	MajorIDs              *[]string `json:"major_ids,omitempty"              validate:"omitempty,dive,uuid"`
	StudyFormatIDs        *[]string `json:"study_format_ids,omitempty"        validate:"omitempty,dive,uuid"`
	SpecialAffiliationIDs *[]string `json:"special_affiliation_ids,omitempty" validate:"omitempty,dive,uuid"`
	AthleticIDs           *[]string `json:"athletic_ids,omitempty"           validate:"omitempty,dive,uuid"`
	SupportServiceIDs     *[]string `json:"support_service_ids,omitempty"     validate:"omitempty,dive,uuid"`
}

type AllLookupsResponse struct {
	Majors              []MajorResponse              `json:"majors"`
	DegreeLevels        []DegreeLevelResponse        `json:"degree_levels"`
	StudyFormats        []StudyFormatResponse        `json:"study_formats"`
	SpecialAffiliations []SpecialAffiliationResponse `json:"special_affiliations"`
	Athletics           []AthleticResponse           `json:"athletics"`
	SupportServices     []SupportServiceResponse     `json:"support_services"`
	Programs            []programs.ProgramLookupItem `json:"programs"`
}

type UniversityListItem struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Slug              string  `json:"slug"`
	Country           string  `json:"country"`
	State             string  `json:"state"`
	City              string  `json:"city"`
	Logo              string  `json:"logo"`
	CoverImage        string  `json:"cover_image"`
	InstitutionType   string  `json:"institution_type"`
	CampusSetting     string  `json:"campus_setting"`
	TuitionMin        int32   `json:"tuition_min"`
	TuitionMax        int32   `json:"tuition_max"`
	AcceptanceRate    float64 `json:"acceptance_rate"`
	IsPopular         bool    `json:"is_popular"`
	IsFeatured        bool    `json:"is_featured"`
	IsFavorited       bool    `json:"is_favorited"`
	HasRepresentative bool    `json:"has_representative"`
}

type UniversityDetailResponse struct {
	CreateUniversityResponse
	DegreeLevels        []DegreeLevelResponse        `json:"degree_levels"`
	Majors              []MajorResponse              `json:"majors"`
	StudyFormats        []StudyFormatResponse        `json:"study_formats"`
	SpecialAffiliations []SpecialAffiliationResponse `json:"special_affiliations"`
	Athletics           []AthleticResponse           `json:"athletics"`
	SupportServices     []SupportServiceResponse     `json:"support_services"`
	HasRepresentative   bool                         `json:"has_representative"`
}

type UniversitySearchResult struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	Country           string `json:"country"`
	State             string `json:"state"`
	City              string `json:"city"`
	FullLocation      string `json:"full_location"`
	Logo              string `json:"logo"`
	IsFavorited       bool   `json:"is_favorited"`
	HasRepresentative bool   `json:"has_representative"`
}

type StatsResponse struct {
	TotalUniversities int64 `json:"total_universities" example:"247"`
	TotalCountries    int64 `json:"total_countries" example:"12"`
	TotalFeatured     int64 `json:"total_featured" example:"18"`
	TotalPopular      int64 `json:"total_popular" example:"24"`
}

type CreateUniversityResponse struct {
	ID                       string    `json:"id"`
	Name                     string    `json:"name"`
	Slug                     string    `json:"slug"`
	Overview                 string    `json:"overview"`
	Excerpt                  string    `json:"excerpt"`
	Country                  string    `json:"country"`
	State                    string    `json:"state"`
	City                     string    `json:"city"`
	FullLocation             string    `json:"full_location"`
	CoverImage               string    `json:"cover_image"`
	Logo                     string    `json:"logo"`
	InstitutionType          string    `json:"institution_type"`
	CampusSetting            string    `json:"campus_setting"`
	InStateTuition           float64   `json:"in_state_tuition"`
	OutOfStateTuition        float64   `json:"out_of_state_tuition"`
	InternationalTuition     float64   `json:"international_tuition"`
	NeedBasedAid             bool      `json:"need_based_aid"`
	MeritScholarships        bool      `json:"merit_scholarships"`
	WorkStudy                bool      `json:"work_study"`
	NoApplicationFee         bool      `json:"no_application_fee"`
	AcceptanceRate           float64   `json:"acceptance_rate"`
	TestingPolicy            string    `json:"testing_policy"`
	SatRange                 string    `json:"sat_range"`
	ActRange                 string    `json:"act_range"`
	OnCampusHousing          bool      `json:"on_campus_housing"`
	FreshmenRequiredOnCampus bool      `json:"freshmen_required_on_campus"`
	ContactEmail             string    `json:"contact_email"`
	ContactPhone             string    `json:"contact_phone"`
	Website                  string    `json:"website"`
	Zipcode                  string    `json:"zipcode"`
	TuitionMin               int32     `json:"tuition_min"`
	TuitionMax               int32     `json:"tuition_max"`
	AvgHighSchoolGpa         float64   `json:"avg_high_school_gpa"`
	FoundedYear              int32     `json:"founded_year"`
	CampusSize               string    `json:"campus_size"`
	GalleryImages            []string  `json:"gallery_images"`
	IsPopular                bool      `json:"is_popular"`
	IsFeatured               bool      `json:"is_featured"`
	MapsUrl                  string    `json:"maps_url"`
	FullAddress              string    `json:"full_address"`
	EmploymentRate           float64   `json:"employment_rate"`
	ResearchOutput           string    `json:"research_output"`
	HousingType              string    `json:"housing_type"`
	SeoTitle                 string    `json:"seo_title"`
	SeoDescription           string    `json:"seo_description"`
	Status                   string    `json:"status" example:"published"`
	PublishedAt              *time.Time `json:"published_at" swaggertype:"string" example:"2026-07-30T12:00:00Z"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}
