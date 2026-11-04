package nlsearch

import "slices"

// The vocabulary is the contract between the model and the database.
//
// Every attribute a user can ask about appears exactly once here, with a
// human-written label that the model reads. The label is what makes
// "located in Kathmandu" map to `city` instead of `country`; the column is
// what the compiler binds against. Both the prompt and the allowlist are
// generated from this file, so a field can never be visible to the model but
// invisible to the validator (or vice versa).

// Kind is the value shape a filter field accepts. It decides which operators
// are legal and how the compiler binds the value.
type Kind string

const (
	KindText   Kind = "text"   // free text, matched with trigram similarity
	KindString Kind = "string" // exact string
	KindNumber Kind = "number"
	KindBool   Kind = "bool"
)

// Op is a comparison operator.
type Op string

const (
	OpEq       Op = "="
	OpNe       Op = "!="
	OpGt       Op = ">"
	OpGte      Op = ">="
	OpLt       Op = "<"
	OpLte      Op = "<="
	OpIn       Op = "in"
	OpContains Op = "contains"
	OpMatches  Op = "matches"
)

// Lookup redirects a many-valued facet to a reference table. The field's
// value is the reference row's name, and the compiler emits an EXISTS over
// the junction table.
type Lookup struct {
	// JoinTable links the owner to the reference rows.
	JoinTable string
	// LookupTable holds the reference rows; its name column is the value.
	LookupTable string
	// FKColumn is the junction column pointing at the reference row.
	FKColumn string
	// ParentColumn is the junction column pointing back at the owning
	// entity, for example "university_id" in university_degree_levels.
	ParentColumn string
}

// Field is one searchable attribute.
type Field struct {
	// Name is what the model emits.
	Name string
	// Label is the description the model reads. Write it the way a person
	// would describe the attribute to another person.
	Label string
	Kind  Kind
	// Ops lists the legal operators for this field.
	Ops []Op
	// Column is the SQL expression the value is compared against.
	Column string
	// Lookup, when set, turns the field into an EXISTS over a reference
	// table instead of a plain column comparison.
	Lookup *Lookup
	// Values is a closed vocabulary. When set, the model may only pick from
	// it and the server rejects anything else.
	Values []string
	// Sortable marks a field the results may be ordered by.
	Sortable bool
}

// Allows reports whether op is legal for this field.
func (f Field) Allows(op Op) bool {
	return slices.Contains(f.Ops, op)
}

// Entity is a searchable table.
type Entity struct {
	// Name is the model-facing entity name.
	Name string
	// Label describes the entity to the model.
	Label string
	// Description says what this catalogue holds. It is what the model reads
	// when it decides which catalogue a question is about.
	Description string
	// Table and Alias are used by the compiler to build SQL.
	Table string
	Alias string
	// StatusColumn holds the publication state. The compiler always restricts
	// results to published rows, so the model can never surface a draft.
	StatusColumn string
	// DefaultOrder is the ORDER BY used when the model asks for no order.
	DefaultOrder string
	// Fields are the attributes searchable on this entity.
	Fields []Field
	// Related maps a related-entity name (e.g. "scholarships") to the
	// relation that reaches it.
	Related map[string]*Relation
}

// Relation lets a filter on this entity reach into another one: "universities
// that offer a scholarship of at least 500000".
type Relation struct {
	// Entity is the related entity's own field set.
	Entity *Entity
	// FK is the column on the related table pointing back at the parent.
	FK string
}

// Field looks up a field by name.
func (e *Entity) Field(name string) (Field, bool) {
	for _, f := range e.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}

// FieldNames returns the model's field names in declaration order.
func (e *Entity) FieldNames() []string {
	names := make([]string, 0, len(e.Fields))
	for _, f := range e.Fields {
		names = append(names, f.Name)
	}
	return names
}

// Closed vocabularies. These mirror the seeded lookup data; keep them in sync
// with the *_seed_*.up.sql migrations and with internal/university/filters.go.

var degreeLevelNames = []string{
	"Certificate",
	"Associate",
	"Bachelor's",
	"Master's",
	"Doctorate",
}

var majorNames = []string{
	"Computer Science",
	"Business",
	"Engineering",
	"Medicine",
	"Biology",
	"Psychology",
	"Economics",
	"Art & Design",
	"Law",
	"Nursing",
}

var studyFormatNames = []string{
	"100% In-Person",
	"100% Online",
	"Hybrid / Blended",
}

var specialAffiliationNames = []string{
	"HBCU",
	"HSI",
	"Women's College",
	"Men's College",
}

var athleticNames = []string{
	"NCAA Division I",
	"NCAA Division II",
	"NCAA Division III",
	"NAIA",
	"Intramural Sports",
}

var campusSettingNames = []string{"Urban", "Suburban", "Rural"}

var institutionTypeNames = []string{
	"Public",
	"Private (Non-Profit)",
	"Private (For-Profit)",
	"2-Year",
	"4-Year",
}

var testingPolicyNames = []string{"Optional", "Blind", "Required"}

var transcriptRequirementNames = []string{"none", "official", "unofficial"}

var educationLevelNames = []string{
	"High School Senior",
	"Undergraduate",
	"Graduate",
	"Ph.D.",
}

var demographicNames = []string{
	"First-Generation",
	"Women",
	"Minority / BIPOC",
	"Military / Veteran",
	"LGBTQ+",
	"Students with Disabilities",
	"International Students",
}

// scholarshipEntity is the related entity reachable from universities and
// colleges. It deliberately omits provider contact details and internal
// notes: nothing outside this file may be reached from a model response.
var scholarshipEntity = &Entity{
	Name:         "scholarships",
	Label:        "a scholarship programme",
	Description:  "scholarship programmes: how much they award, who is eligible, when to apply, and the organisation offering them",
	Table:        "scholarships",
	Alias:        "s",
	StatusColumn: "status",
	DefaultOrder: "s.created_at DESC, s.title ASC",
	Fields: []Field{
		{
			Name:   "title",
			Label:  "the name of the scholarship programme",
			Kind:   KindText,
			Ops:    []Op{OpMatches, OpContains},
			Column: "s.title",
		},
		{
			Name:   "provider_name",
			Label:  "the organisation offering the scholarship",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpContains},
			Column: "s.provider_name",
		},
		{
			Name:   "provider_type",
			Label:  "the type of provider offering the scholarship, for example Government, Corporation, NGO or Foundation",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe},
			Column: "s.provider_type",
		},
		{
			Name:   "country",
			Label:  "the country the scholarship is available in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "s.country",
		},
		{
			Name:   "state",
			Label:  "the state, province or region the scholarship is available in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "s.state",
		},
		{
			Name:   "city",
			Label:  "the city the scholarship is available in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "s.city",
		},
		{
			Name:   "award_min",
			Label:  "the smallest award amount the scholarship pays, in the local currency",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "s.award_min",
		},
		{
			Name:   "award_max",
			Label:  "the largest award amount the scholarship pays, in the local currency",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "s.award_max",
		},
		{
			Name:   "number_of_awards",
			Label:  "how many awards the scholarship gives out",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "s.number_of_awards",
		},
		{
			Name:   "min_gpa",
			Label:  "the minimum GPA a student needs to qualify for the scholarship",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "s.min_gpa",
		},
		{
			Name:   "is_renewable",
			Label:  "true when the scholarship can be renewed every year",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "s.is_renewable",
		},
		{
			Name:   "requires_financial_need",
			Label:  "true when the scholarship is only for students who need financial aid",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "s.requires_financial_need",
		},
		{
			Name:   "essay_required",
			Label:  "true when applying for the scholarship requires an essay",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "s.essay_required",
		},
		{
			Name:   "recommendation_letters_required",
			Label:  "how many recommendation letters the application needs",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "s.recommendation_letters_required",
		},
		{
			Name:   "portfolio_required",
			Label:  "true when applying for the scholarship requires a portfolio",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "s.portfolio_required",
		},
		{
			Name:   "transcript_requirement",
			Label:  "how much official documentation the application needs",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe},
			Column: "s.transcript_requirement",
			Values: transcriptRequirementNames,
		},
		{
			Name:   "is_popular",
			Label:  "true when the scholarship is marked as popular",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "s.is_popular",
		},
		{
			Name:   "is_featured",
			Label:  "true when the scholarship is marked as featured",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "s.is_featured",
		},
		{
			Name:   "education_levels",
			Label:  "the education levels the scholarship is open to",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "scholarship_education_levels",
				LookupTable:  "education_levels",
				FKColumn:     "education_level_id",
				ParentColumn: "scholarship_id",
			},
			Values: educationLevelNames,
		},
		{
			Name:   "majors",
			Label:  "the fields of study the scholarship is open to",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "scholarship_majors",
				LookupTable:  "majors",
				FKColumn:     "major_id",
				ParentColumn: "scholarship_id",
			},
			Values: majorNames,
		},
		{
			Name:   "demographics",
			Label:  "the groups of students the scholarship is open to",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "scholarship_demographics",
				LookupTable:  "demographics",
				FKColumn:     "demographic_id",
				ParentColumn: "scholarship_id",
			},
			Values: demographicNames,
		},
	},
}

var universityEntity = &Entity{
	Name:         "universities",
	Label:        "a university",
	Description:  "universities and degree-awarding institutions: where they are, what they charge, how selective they are, the degrees and majors they teach, and the scholarships they offer",
	Table:        "universities",
	Alias:        "u",
	StatusColumn: "status",
	DefaultOrder: "u.name ASC",
	Fields: []Field{
		{
			Name:   "name",
			Label:  "the name of the university",
			Kind:   KindText,
			Ops:    []Op{OpMatches, OpContains},
			Column: "u.name",
		},
		{
			Name:   "country",
			Label:  "the country the university is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn, OpContains},
			Column: "u.country",
		},
		{
			Name:   "state",
			Label:  "the state, province or region the university is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn, OpContains},
			Column: "u.state",
		},
		{
			Name:   "city",
			Label:  "the city the university is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn, OpContains},
			Column: "u.city",
		},
		{
			Name:   "continent",
			Label:  "the continent the university is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "u.continent",
		},
		{
			Name:   "full_location",
			Label:  "the full written location of the university, for example 'Dhulikhel, Bagmati, Nepal'",
			Kind:   KindText,
			Ops:    []Op{OpContains, OpMatches},
			Column: "u.full_location",
		},
		{
			Name:   "institution_type",
			Label:  "whether the university is public or private",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "u.institution_type",
			Values: institutionTypeNames,
		},
		{
			Name:   "campus_setting",
			Label:  "whether the campus is in a city, a suburb or a rural area",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "u.campus_setting",
			Values: campusSettingNames,
		},
		{
			Name:   "testing_policy",
			Label:  "whether SAT or ACT scores are required, optional or not considered",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "u.testing_policy",
			Values: testingPolicyNames,
		},
		{
			Name:   "tuition_min",
			Label:  "the lowest annual tuition the university charges",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "u.tuition_min",
		},
		{
			Name:   "tuition_max",
			Label:  "the highest annual tuition the university charges",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "u.tuition_max",
		},
		{
			Name:   "in_state_tuition",
			Label:  "the annual tuition for students from the same state or country",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "u.in_state_tuition",
		},
		{
			Name:   "out_of_state_tuition",
			Label:  "the annual tuition for students from outside the state or country",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "u.out_of_state_tuition",
		},
		{
			Name:   "international_tuition",
			Label:  "the annual tuition for international students",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "u.international_tuition",
		},
		{
			Name:   "acceptance_rate",
			Label:  "the percentage of applicants the university accepts",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "u.acceptance_rate",
		},
		{
			Name:   "employment_rate",
			Label:  "the percentage of graduates who find work, in percent",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "u.employment_rate",
		},
		{
			Name:   "avg_high_school_gpa",
			Label:  "the average high school GPA of admitted students",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "u.avg_high_school_gpa",
		},
		{
			Name:   "founded_year",
			Label:  "the year the university was founded",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "u.founded_year",
		},
		{
			Name:   "need_based_aid",
			Label:  "true when the university offers financial aid to students who need it",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "u.need_based_aid",
		},
		{
			Name:   "merit_scholarships",
			Label:  "true when the university awards scholarships based on academic merit",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "u.merit_scholarships",
		},
		{
			Name:   "work_study",
			Label:  "true when the university offers work-study programmes",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "u.work_study",
		},
		{
			Name:   "no_application_fee",
			Label:  "true when the university charges no application fee",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "u.no_application_fee",
		},
		{
			Name:   "on_campus_housing",
			Label:  "true when the university offers on-campus housing",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "u.on_campus_housing",
		},
		{
			Name:   "freshmen_required_on_campus",
			Label:  "true when first-year students must live on campus",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "u.freshmen_required_on_campus",
		},
		{
			Name:   "is_popular",
			Label:  "true when the university is marked as popular",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "u.is_popular",
		},
		{
			Name:   "is_featured",
			Label:  "true when the university is marked as featured",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "u.is_featured",
		},
		{
			Name:   "research_output",
			Label:  "the research rating of the university, for example R1 or R2",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpContains},
			Column: "u.research_output",
		},
		{
			Name:   "degree_levels",
			Label:  "the degree levels the university awards, for example Bachelor's or Master's",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "university_degree_levels",
				LookupTable:  "degree_levels",
				FKColumn:     "degree_level_id",
				ParentColumn: "university_id",
			},
			Values: degreeLevelNames,
		},
		{
			Name:   "majors",
			Label:  "the fields of study the university offers, for example Engineering or Business",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "university_majors",
				LookupTable:  "majors",
				FKColumn:     "major_id",
				ParentColumn: "university_id",
			},
			Values: majorNames,
		},
		{
			Name:   "study_formats",
			Label:  "how the university delivers its courses, for example fully online or fully in person",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "university_study_formats",
				LookupTable:  "study_formats",
				FKColumn:     "study_format_id",
				ParentColumn: "university_id",
			},
			Values: studyFormatNames,
		},
		{
			Name:   "special_affiliations",
			Label:  "the special affiliation the university belongs to, for example HBCU or a women's college",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "university_special_affiliations",
				LookupTable:  "special_affiliations",
				FKColumn:     "special_affiliation_id",
				ParentColumn: "university_id",
			},
			Values: specialAffiliationNames,
		},
		{
			Name:   "athletics",
			Label:  "the athletic programmes the university competes in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "university_athletics",
				LookupTable:  "athletics",
				FKColumn:     "athletic_id",
				ParentColumn: "university_id",
			},
			Values: athleticNames,
		},
	},
}

var collegeEntity = &Entity{
	Name:         "colleges",
	Label:        "a college",
	Description:  "colleges: where they are, the degrees and majors they teach, and the scholarships they offer",
	Table:        "colleges",
	Alias:        "c",
	StatusColumn: "status",
	DefaultOrder: "c.name ASC",
	Fields: []Field{
		{
			Name:   "name",
			Label:  "the name of the college",
			Kind:   KindText,
			Ops:    []Op{OpMatches, OpContains},
			Column: "c.name",
		},
		{
			Name:   "country",
			Label:  "the country the college is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn, OpContains},
			Column: "c.country",
		},
		{
			Name:   "state",
			Label:  "the state, province or region the college is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn, OpContains},
			Column: "c.state",
		},
		{
			Name:   "city",
			Label:  "the city the college is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn, OpContains},
			Column: "c.city",
		},
		{
			Name:   "continent",
			Label:  "the continent the college is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "c.continent",
		},
		{
			Name:   "full_location",
			Label:  "the full written location of the college, for example 'Lalitpur, Bagmati, Nepal'",
			Kind:   KindText,
			Ops:    []Op{OpContains, OpMatches},
			Column: "c.full_location",
		},
		{
			Name:   "institution_type",
			Label:  "whether the college is public or private",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "c.institution_type",
			Values: institutionTypeNames,
		},
		{
			Name:   "campus_setting",
			Label:  "whether the campus is in a city, a suburb or a rural area",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "c.campus_setting",
			Values: campusSettingNames,
		},
		{
			Name:   "founded_year",
			Label:  "the year the college was founded",
			Kind:   KindNumber,
			Ops:    []Op{OpEq, OpGte, OpGt, OpLte, OpLt},
			Column: "c.founded_year",
		},
		{
			Name:   "campus_size",
			Label:  "the size of the campus, for example '12 acres'",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpContains},
			Column: "c.campus_size",
		},
		{
			Name:   "is_popular",
			Label:  "true when the college is marked as popular",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "c.is_popular",
		},
		{
			Name:   "is_featured",
			Label:  "true when the college is marked as featured",
			Kind:   KindBool,
			Ops:    []Op{OpEq},
			Column: "c.is_featured",
		},
		{
			Name:   "degree_levels",
			Label:  "the degree levels the college awards, for example Bachelor's or Master's",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "college_degree_levels",
				LookupTable:  "degree_levels",
				FKColumn:     "degree_level_id",
				ParentColumn: "college_id",
			},
			Values: degreeLevelNames,
		},
		{
			Name:   "majors",
			Label:  "the fields of study the college offers, for example Engineering or Business",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "college_majors",
				LookupTable:  "majors",
				FKColumn:     "major_id",
				ParentColumn: "college_id",
			},
			Values: majorNames,
		},
		{
			Name:   "study_formats",
			Label:  "how the college delivers its courses, for example fully online or fully in person",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "college_study_formats",
				LookupTable:  "study_formats",
				FKColumn:     "study_format_id",
				ParentColumn: "college_id",
			},
			Values: studyFormatNames,
		},
	},
}

// sortableFields lists the attributes each entity's results may be ordered
// by. Everything else is rejected when the model asks to sort by it.
var sortableFields = map[string][]string{
	"universities": {
		"name", "tuition_min", "tuition_max", "acceptance_rate",
		"employment_rate", "founded_year", "is_popular", "is_featured",
	},
	"colleges": {
		"name", "founded_year", "is_popular", "is_featured",
	},
	"scholarships": {
		"title", "award_min", "award_max", "number_of_awards", "min_gpa",
		"is_popular", "is_featured",
	},
}

func init() {
	universityEntity.Related = map[string]*Relation{
		"scholarships": {Entity: scholarshipEntity, FK: "university_id"},
	}
	collegeEntity.Related = map[string]*Relation{
		"scholarships": {Entity: scholarshipEntity, FK: "college_id"},
	}
	for _, entity := range []*Entity{universityEntity, collegeEntity, scholarshipEntity} {
		for _, name := range sortableFields[entity.Name] {
			for i := range entity.Fields {
				if entity.Fields[i].Name == name {
					entity.Fields[i].Sortable = true
				}
			}
		}
	}
}

// entities is the model-facing catalogue. The order is the order the model
// sees, so the most commonly searched entity comes first.
var entities = []*Entity{
	universityEntity,
	collegeEntity,
	scholarshipEntity,
}

// entityByName resolves a model-supplied entity name.
func entityByName(name string) (*Entity, bool) {
	for _, e := range entities {
		if e.Name == name {
			return e, true
		}
	}
	return nil, false
}
