package university

import "time"

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
	Name     string `json:"name" example:"Massachusetts Institute of Technology"`
	Slug     string `json:"slug" example:"mit"`
	Overview string `json:"overview" example:"MIT is a private research university in Cambridge, Massachusetts."`
	Excerpt  string `json:"excerpt" example:"World-class research university founded in 1861."`

	Country      string `json:"country" example:"US"`
	State        string `json:"state" example:"MA"`
	City         string `json:"city" example:"Cambridge"`
	FullLocation string `json:"full_location" example:"Cambridge, MA, US"`

	CoverImage string `json:"cover_image" example:"https://cdn.example.com/mit-cover.jpg"`
	Logo       string `json:"logo" example:"https://cdn.example.com/mit-logo.png"`

	InstitutionType string `json:"institution_type" example:"Private"`
	CampusSetting   string `json:"campus_setting" example:"Urban"`

	InStateTuition       float64 `json:"in_state_tuition" example:"57590"`
	OutOfStateTuition    float64 `json:"out_of_state_tuition" example:"57590"`
	InternationalTuition float64 `json:"international_tuition" example:"57590"`

	NeedBasedAid      bool `json:"need_based_aid" example:"true"`
	MeritScholarships bool `json:"merit_scholarships" example:"true"`
	WorkStudy         bool `json:"work_study" example:"true"`
	NoApplicationFee  bool `json:"no_application_fee" example:"false"`

	AcceptanceRate float64 `json:"acceptance_rate" example:"0.04"`
	TestingPolicy  string  `json:"testing_policy" example:"Optional"`
	SatRange       string  `json:"sat_range" example:"1500-1570"`
	ActRange       string  `json:"act_range" example:"34-36"`

	OnCampusHousing          bool `json:"on_campus_housing" example:"true"`
	FreshmenRequiredOnCampus bool `json:"freshmen_required_on_campus" example:"true"`

	ContactEmail string `json:"contact_email" example:"admissions@mit.edu"`
	ContactPhone string `json:"contact_phone" example:"+1-617-253-1000"`
	Website      string `json:"website" example:"https://www.mit.edu"`

	DegreeLevelIDs        []string `json:"degree_level_ids" validate:"dive,uuid"`
	MajorIDs              []string `json:"major_ids" validate:"dive,uuid"`
	StudyFormatIDs        []string `json:"study_format_ids" validate:"dive,uuid"`
	SpecialAffiliationIDs []string `json:"special_affiliation_ids" validate:"dive,uuid"`
	AthleticIDs           []string `json:"athletic_ids" validate:"dive,uuid"`
	SupportServiceIDs     []string `json:"support_service_ids" validate:"dive,uuid"`
}

type AllLookupsResponse struct {
	Majors              []MajorResponse              `json:"majors"`
	DegreeLevels        []DegreeLevelResponse        `json:"degree_levels"`
	StudyFormats        []StudyFormatResponse        `json:"study_formats"`
	SpecialAffiliations []SpecialAffiliationResponse `json:"special_affiliations"`
	Athletics           []AthleticResponse           `json:"athletics"`
	SupportServices     []SupportServiceResponse     `json:"support_services"`
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
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}
