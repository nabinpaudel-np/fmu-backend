package nlsearch

import (
	"slices"
	"strings"
)

// The vocabulary is the contract between the model and the database. Every
// attribute a user can ask about appears exactly once here, with a label the
// model reads. The prompt and the validator are both generated from this
// file, so they cannot drift apart.

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

// Lookup redirects a many-valued facet to a reference table.
type Lookup struct {
	JoinTable    string
	LookupTable  string
	FKColumn     string
	ParentColumn string
}

// Code names the abbreviation a column holds. A question names a country in
// words, so the model's value is translated into the code before it is bound.
type Code string

const (
	// CodeCountry is an ISO 3166-1 alpha-2 country code.
	CodeCountry Code = "country"
	// CodeState is a two-letter US state code.
	CodeState Code = "state"
)

// Field is one searchable attribute.
type Field struct {
	Name         string
	Label        string
	Kind         Kind
	Ops          []Op
	Column       string
	Lookup       *Lookup
	Values       []string
	CommonValues []string
	Code         Code
	Sortable     bool
}

// Quality is a judgement the request makes about a field rather than a fact
// it states: "affordable", "selective", "prestigious". The model cannot
// derive a threshold from the word alone, so each quality carries the
// threshold to use, and the prompt renders them for the model.
//
// A quality ranks by default. "Affordable universities" means the affordable
// ones, ordered cheapest first — not a cut-off that silently returns nothing
// when the data does not fit under it. The tuition in this catalogue runs
// from 0 to 9,000,000 across currencies, so a fixed cut-off would drop every
// US university.
//
// A quality that names a number is different: "selective" means an
// acceptance rate under a threshold, and a row outside it is not selective.
// Those set Threshold, and they filter as well as rank.
type Quality struct {
	Word      string
	Field     string
	Op        Op
	Value     float64
	Sort      bool
	Threshold bool
	Ascending bool
	Rationale string
}

// Allows reports whether op is legal for this field.
func (f Field) Allows(op Op) bool {
	return slices.Contains(f.Ops, op)
}

// Entity is a searchable table.
type Entity struct {
	Name         string
	Label        string
	Table        string
	Alias        string
	StatusColumn string
	DefaultOrder string
	Fields       []Field
	Related      map[string]*Relation

	// Content names the columns a free-text question is matched against.
	// The vector column holds the indexed text.
	Content []ContentField
}

// ContentField is one column of an entity's searchable prose.
type ContentField struct {
	Label  string
	Column string
}

// Relation lets a filter on this entity reach into another one: "universities
// that offer a scholarship of at least 500000".
type Relation struct {
	Entity  *Entity
	FK      string
	Through *Through
}

// Through describes a link that runs through a shared middle table.
type Through struct {
	Table string

	ParentJunction string
	ParentColumn   string
	MiddleColumn   string

	EntityJunction     string
	EntityColumn       string
	MiddleEntityColumn string
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

var campusSettingNames = []string{"urban", "suburban", "rural"}

// These three sit in free-text columns, so the values are only guidance.
var institutionTypeNames = []string{
	"public",
	"private",
	"2-year",
	"4-year",
}

var testingPolicyNames = []string{
	"test-optional",
	"test-blind",
	"test-required",
	"not required",
}

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

// Qualities are the judgements a request makes instead of stating a number:
// "affordable", "selective", "old". The model cannot work out a threshold
// from the word on its own, so each one carries how it should be read.
//
// The money qualities rank rather than filter. Tuition in this catalogue runs
// from 0 to 9,000,000 and is in the local currency of each row's country, so
// a fixed cut-off would be meaningless — it would return nothing for a
// country whose fees are all above it. "Affordable universities" therefore
// means the universities ordered cheapest first, and the reader can see the
// prices.
//
// The qualities that do have a natural line — a selective school, an old
// one — set a threshold, and a row outside it is not that thing at all.
var qualities = []Quality{
	{
		Word:      "affordable",
		Field:     "tuition_max",
		Op:        OpLte,
		Sort:      true,
		Ascending: true,
		Rationale: "ordered by tuition, cheapest first",
	},
	{
		Word:      "cheap",
		Field:     "tuition_max",
		Op:        OpLte,
		Sort:      true,
		Ascending: true,
		Rationale: "ordered by tuition, cheapest first",
	},
	{
		Word:      "budget",
		Field:     "tuition_max",
		Op:        OpLte,
		Sort:      true,
		Ascending: true,
		Rationale: "ordered by tuition, cheapest first",
	},
	{
		Word:      "inexpensive",
		Field:     "tuition_max",
		Op:        OpLte,
		Sort:      true,
		Ascending: true,
		Rationale: "ordered by tuition, cheapest first",
	},
	{
		Word:      "low cost",
		Field:     "tuition_max",
		Op:        OpLte,
		Sort:      true,
		Ascending: true,
		Rationale: "ordered by tuition, cheapest first",
	},
	{
		Word:      "expensive",
		Field:     "tuition_min",
		Op:        OpGte,
		Sort:      true,
		Ascending: false,
		Rationale: "ordered by tuition, most expensive first",
	},
	{
		Word:      "selective",
		Field:     "acceptance_rate",
		Op:        OpLt,
		Ascending: true,
		Value:     30,
		Sort:      true,
		Threshold: true,
		Rationale: "a university is selective when it accepts fewer than 30 percent of applicants",
	},
	{
		Word:      "prestigious",
		Field:     "acceptance_rate",
		Op:        OpLt,
		Ascending: true,
		Value:     20,
		Sort:      true,
		Threshold: true,
		Rationale: "a university is prestigious when it accepts fewer than 20 percent of applicants",
	},
	{
		Word:      "old",
		Field:     "founded_year",
		Op:        OpLt,
		Ascending: true,
		Value:     1950,
		Sort:      true,
		Threshold: true,
		Rationale: "a university is old when it was founded before 1950",
	},
	{
		Word:      "established",
		Field:     "founded_year",
		Op:        OpLt,
		Ascending: true,
		Value:     1950,
		Sort:      true,
		Threshold: true,
		Rationale: "a university is established when it was founded before 1950",
	},
}

// MatchQuality finds the quality a request names, if it names one. The
// question is matched on its words, so a phrase like "low cost" is found
// before the single word "low" would be.
//
// The entity matters: a quality is only offered for a catalogue that has the
// field it refers to. "affordable" needs a tuition field, so it is not
// offered for scholarships.
func MatchQuality(entity *Entity, question string) (Quality, bool) {
	if entity == nil {
		return Quality{}, false
	}
	lowered := strings.ToLower(question)
	for _, q := range qualities {
		if !containsPhrase(lowered, q.Word) {
			continue
		}
		if precededBySuperlative(lowered, q.Word) {
			continue
		}
		field, ok := entity.Field(q.Field)
		if !ok || !field.Allows(q.Op) {
			continue
		}
		return q, true
	}
	return Quality{}, false
}

// superlativeMarkers turn a quality into an ordering. Only the words that
// already name an extreme do this: "least expensive" asks for the cheapest
// one, which is a sort, while "most affordable" still asks for a set of
// affordable universities, which is a filter.
//
// The marker has to sit directly before the quality word, and "most" is not
// one of them: "most affordable" is a set, "least affordable" is not.
var superlativeMarkers = []string{"least", "cheapest", "lowest", "highest", "biggest", "largest", "smallest"}

// precededBySuperlative reports whether the word before the phrase makes the
// phrase an ordering rather than a threshold.
func precededBySuperlative(text, phrase string) bool {
	offset := 0
	for {
		i := strings.Index(text[offset:], phrase)
		if i < 0 {
			return false
		}
		i += offset
		before := i == 0 || !isWordByte(text[i-1])
		after := i+len(phrase) == len(text) || !isWordByte(text[i+len(phrase)])
		if before && after {
			prefix := strings.TrimSpace(text[:i])
			for _, marker := range superlativeMarkers {
				if strings.HasSuffix(prefix, marker) {
					return true
				}
			}
			return false
		}
		offset = i + len(phrase)
	}
}

// containsPhrase reports whether a phrase appears in the text as whole words.
// A plain substring test would match "cheap" inside "cheapest", and the two
// mean different things: "cheap" is a threshold, "cheapest" is an ordering.
func containsPhrase(text, phrase string) bool {
	if phrase == "" {
		return false
	}
	offset := 0
	for {
		i := strings.Index(text[offset:], phrase)
		if i < 0 {
			return false
		}
		i += offset
		before := i == 0 || !isWordByte(text[i-1])
		after := i+len(phrase) == len(text) || !isWordByte(text[i+len(phrase)])
		if before && after {
			return true
		}
		offset = i + len(phrase)
	}
}

// isWordByte reports whether a byte is part of a word. Anything that is not a
// letter, a digit or an underscore is a boundary.
func isWordByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	case b == '_':
		return true
	}
	return false
}

// scholarshipEntity is the related entity reachable from universities and
var scholarshipEntity = &Entity{
	Name:         "scholarships",
	Label:        "a scholarship programme",
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
			Code:   CodeCountry,
		},
		{
			Name:   "state",
			Label:  "the state, province or region the scholarship is available in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "s.state",
			Code:   CodeState,
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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

// programEntity is the catalogue of study programmes.
var programEntity = &Entity{
	Name:  "programs",
	Label: "a study programme",
	// "degree" and "bachelor"/"master" are deliberately not keywords here.
	// They are qualifiers far more often than they name a catalogue — a
	// question about "bachelor's degree colleges" is about colleges — and
	// the model can still reach a programme through the related catalogues.
	Table:        "programs",
	Alias:        "p",
	StatusColumn: "",
	DefaultOrder: "p.title ASC",
	Fields: []Field{
		{
			Name:   "title",
			Label:  "the name of the programme, for example BCA or MBA",
			Kind:   KindText,
			Ops:    []Op{OpMatches, OpContains},
			Column: "p.title",
		},
		{
			Name:   "description",
			Label:  "what the programme covers",
			Kind:   KindText,
			Ops:    []Op{OpContains, OpMatches},
			Column: "p.description",
		},
		{
			Name:   "excerpt",
			Label:  "the short summary of the programme",
			Kind:   KindText,
			Ops:    []Op{OpContains, OpMatches},
			Column: "p.excerpt",
		},
		{
			Name:   "career_options",
			Label:  "the careers the programme leads to, for example software developer or data analyst",
			Kind:   KindText,
			Ops:    []Op{OpContains, OpMatches},
			Column: "p.career_options",
		},
		{
			Name:   "degree_levels",
			Label:  "the degree level the programme belongs to, for example Bachelor's or Master's",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn},
			Column: "",
			Lookup: &Lookup{
				JoinTable:    "programs",
				LookupTable:  "degree_levels",
				FKColumn:     "degree_id",
				ParentColumn: "id",
			},
			Values: degreeLevelNames,
		},
	},
	// A programme is described by what it covers and what it leads to.
	Content: []ContentField{
		{Label: "what the programme covers", Column: "description"},
		{Label: "the short summary of the programme", Column: "excerpt"},
		{Label: "the careers the programme leads to", Column: "career_options"},
	},
}

var universityEntity = &Entity{
	Name:         "universities",
	Label:        "a university",
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
			Code:   CodeCountry,
		},
		{
			Name:   "state",
			Label:  "the state, province or region the university is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn, OpContains},
			Column: "u.state",
			Code:   CodeState,
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
			Name:         "institution_type",
			Label:        "whether the university is public or private",
			Kind:         KindString,
			Ops:          []Op{OpEq, OpContains, OpNe},
			Column:       "u.institution_type",
			CommonValues: institutionTypeNames,
		},
		{
			Name:         "campus_setting",
			Label:        "whether the campus is in a city, a suburb or a rural area",
			Kind:         KindString,
			Ops:          []Op{OpEq, OpContains, OpNe},
			Column:       "u.campus_setting",
			CommonValues: campusSettingNames,
		},
		{
			Name:         "testing_policy",
			Label:        "whether SAT or ACT scores are required, optional or not considered",
			Kind:         KindString,
			Ops:          []Op{OpEq, OpContains, OpNe},
			Column:       "u.testing_policy",
			CommonValues: testingPolicyNames,
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
	// A university is described by its overview and short summary.
	Content: []ContentField{
		{Label: "the description of the university", Column: "overview"},
		{Label: "the short summary of the university", Column: "excerpt"},
	},
}

var collegeEntity = &Entity{
	Name:         "colleges",
	Label:        "a college",
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
			Code:   CodeCountry,
		},
		{
			Name:   "state",
			Label:  "the state, province or region the college is in",
			Kind:   KindString,
			Ops:    []Op{OpEq, OpNe, OpIn, OpContains},
			Column: "c.state",
			Code:   CodeState,
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
			Name:         "institution_type",
			Label:        "whether the college is public or private",
			Kind:         KindString,
			Ops:          []Op{OpEq, OpContains, OpNe},
			Column:       "c.institution_type",
			CommonValues: institutionTypeNames,
		},
		{
			Name:         "campus_setting",
			Label:        "whether the campus is in a city, a suburb or a rural area",
			Kind:         KindString,
			Ops:          []Op{OpEq, OpContains, OpNe},
			Column:       "c.campus_setting",
			CommonValues: campusSettingNames,
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
			Ops:    []Op{OpEq, OpNe, OpIn},
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
	// A college is described by its overview and short summary.
	Content: []ContentField{
		{Label: "the description of the college", Column: "overview"},
		{Label: "the short summary of the college", Column: "excerpt"},
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
	"programs": {
		"title",
	},
}

func init() {
	universityEntity.Related = map[string]*Relation{
		"scholarships": {Entity: scholarshipEntity, FK: "university_id"},
		"colleges":     {Entity: collegeEntity, FK: "university_id"},
		"programs": {Entity: programEntity, Through: &Through{
			// A university awards degree levels through
			// university_degree_levels, and a programme belongs to a degree
			// level through its own degree_id column. The two meet on the
			// degree_levels row.
			Table:              "degree_levels",
			ParentJunction:     "university_degree_levels",
			ParentColumn:       "university_id",
			MiddleColumn:       "degree_level_id",
			EntityJunction:     "",
			EntityColumn:       "degree_id",
			MiddleEntityColumn: "id",
		}},
	}
	collegeEntity.Related = map[string]*Relation{
		"scholarships": {Entity: scholarshipEntity, FK: "college_id"},
		"universities": {Entity: universityEntity, FK: "university_id"},
		"programs": {Entity: programEntity, Through: &Through{
			Table:              "degree_levels",
			ParentJunction:     "college_degree_levels",
			ParentColumn:       "college_id",
			MiddleColumn:       "degree_level_id",
			EntityJunction:     "",
			EntityColumn:       "degree_id",
			MiddleEntityColumn: "id",
		}},
	}
	scholarshipEntity.Related = map[string]*Relation{
		"universities": {Entity: universityEntity, FK: "university_id"},
		"colleges":     {Entity: collegeEntity, FK: "college_id"},
	}
	programEntity.Related = map[string]*Relation{
		"universities": {Entity: universityEntity, Through: &Through{
			Table:              "degree_levels",
			ParentJunction:     "",
			ParentColumn:       "degree_id",
			MiddleColumn:       "id",
			EntityJunction:     "university_degree_levels",
			EntityColumn:       "university_id",
			MiddleEntityColumn: "degree_level_id",
		}},
		"colleges": {Entity: collegeEntity, Through: &Through{
			Table:              "degree_levels",
			ParentJunction:     "",
			ParentColumn:       "degree_id",
			MiddleColumn:       "id",
			EntityJunction:     "college_degree_levels",
			EntityColumn:       "college_id",
			MiddleEntityColumn: "degree_level_id",
		}},
	}
	for _, entity := range []*Entity{universityEntity, collegeEntity, scholarshipEntity, programEntity} {
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
	programEntity,
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
