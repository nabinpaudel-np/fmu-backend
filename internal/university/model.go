package university

import "time"

type University struct {
	ID       string
	Name     string
	Slug     string
	Overview string
	Excerpt  string

	Country      string
	State        string
	City         string
	FullLocation string

	CoverImage string
	Logo       string

	InstitutionType string
	CampusSetting   string

	InStateTuition       float64
	OutOfStateTuition    float64
	InternationalTuition float64

	NeedBasedAid      bool
	MeritScholarships bool
	WorkStudy         bool
	NoApplicationFee  bool

	AcceptanceRate float64
	TestingPolicy  string
	SatRange       string
	ActRange       string

	OnCampusHousing          bool
	FreshmenRequiredOnCampus bool

	ContactEmail string
	ContactPhone string
	Website      string

	CreatedAt time.Time
	UpdatedAt time.Time
}
