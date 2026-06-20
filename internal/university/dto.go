package university

type CreateUniversityRequest struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Overview string `json:"overview"`
	Excerpt  string `json:"excerpt"`

	Country      string `json:"country"`
	State        string `json:"state"`
	City         string `json:"city"`
	FullLocation string `json:"full_location"`

	CoverImage string `json:"cover_image"`
	Logo       string `json:"logo"`

	InstitutionType string `json:"institution_type"`
	CampusSetting   string `json:"campus_setting"`

	InStateTuition       float64 `json:"in_state_tuition"`
	OutOfStateTuition    float64 `json:"out_of_state_tuition"`
	InternationalTuition float64 `json:"international_tuition"`

	NeedBasedAid      bool `json:"need_based_aid"`
	MeritScholarships bool `json:"merit_scholarships"`
	WorkStudy         bool `json:"work_study"`
	NoApplicationFee  bool `json:"no_application_fee"`

	AcceptanceRate float64 `json:"acceptance_rate"`
	TestingPolicy  string  `json:"testing_policy"`
	SatRange       string  `json:"sat_range"`
	ActRange       string  `json:"act_range"`

	OnCampusHousing          bool `json:"on_campus_housing"`
	FreshmenRequiredOnCampus bool `json:"freshmen_required_on_campus"`

	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	Website      string `json:"website"`

	DegreeLevelIDs        []string `json:"degree_level_ids"`
	MajorIDs              []string `json:"major_ids"`
	StudyFormatIDs        []string `json:"study_format_ids"`
	SpecialAffiliationIDs []string `json:"special_affiliation_ids"`
	AthleticIDs           []string `json:"athletic_ids"`
	SupportServiceIDs     []string `json:"support_service_ids"`
}
