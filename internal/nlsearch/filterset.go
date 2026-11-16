package nlsearch

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// The model speaks in two layers: a JSON DTO it produces, and a validated
// Plan the compiler consumes. The DTO is re-read against the vocabulary
// here, and the compiler only ever sees values that came out of this file.

// FilterDTO is one constraint as the model writes it.
type FilterDTO struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// FilterGroupDTO is a set of constraints of which at least one must hold.
type FilterGroupDTO struct {
	Any []FilterDTO `json:"any"`
}

// ConditionDTO is one constraint in a plan: a plain filter or a group.
type ConditionDTO struct {
	Field string      `json:"field,omitempty"`
	Op    string      `json:"op,omitempty"`
	Value any         `json:"value,omitempty"`
	Any   []FilterDTO `json:"any,omitempty"`
}

// RelatedDTO is a set of constraints on a related entity.
type RelatedDTO struct {
	Entity  string         `json:"entity"`
	Filters []ConditionDTO `json:"filters"`
}

// SortDTO is the requested ordering.
type SortDTO struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// FilterSetDTO is the model's entire output. `assumptions` and `unmapped` are
// shown to the user, never executed.
type FilterSetDTO struct {
	Entity      string         `json:"entity"`
	Filters     []ConditionDTO `json:"filters"`
	Related     []RelatedDTO   `json:"related"`
	Sort        *SortDTO       `json:"sort"`
	Assumptions string         `json:"assumptions"`
	Unmapped    []string       `json:"unmapped"`

	Also []AlsoDTO `json:"also,omitempty"`
}

// AlsoDTO is one extra search over another catalogue.
type AlsoDTO struct {
	Entity      string         `json:"entity"`
	Filters     []ConditionDTO `json:"filters"`
	Related     []RelatedDTO   `json:"related"`
	Sort        *SortDTO       `json:"sort"`
	Assumptions string         `json:"assumptions,omitempty"`
}

// ValueKind is the type of a normalised filter value.
type ValueKind int

const (
	ValueString ValueKind = iota
	ValueNumber
	ValueBool
	ValueList
)

// Value is a normalised filter value. Exactly one member is meaningful,
// selected by Kind.
type Value struct {
	Kind ValueKind
	Str  string
	Num  float64
	Bool bool
	List []string
}

// FilterSpec is a validated constraint.
type FilterSpec struct {
	Field string
	Op    Op
	Value Value
	Any   []FilterSpec
}

// RelatedSpec is a validated set of constraints on a related entity.
type RelatedSpec struct {
	Relation *Relation
	Filters  []FilterSpec
}

// SortSpec is a validated ordering.
type SortSpec struct {
	Field     string
	Direction string
}

// Plan is a validated, normalised filter set ready for the compiler.
type Plan struct {
	Entity      *Entity
	Filters     []FilterSpec
	Related     []RelatedSpec
	Sort        *SortSpec
	Assumptions string
	Unmapped    []string
}

// Plans is a set of validated plans: one per catalogue a question spans.
type Plans struct {
	Primary *Plan
	Others  []*Plan
}

// All returns every plan, the primary first.
func (p *Plans) All() []*Plan {
	if p == nil {
		return nil
	}
	out := make([]*Plan, 0, len(p.Others)+1)
	if p.Primary != nil {
		out = append(out, p.Primary)
	}
	return append(out, p.Others...)
}

// Entities lists the catalogues the plans cover, the primary first.
func (p *Plans) Entities() []string {
	var out []string
	for _, plan := range p.All() {
		out = append(out, plan.Entity.Name)
	}
	return out
}

// ValidationError describes a filter the model got wrong.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

// maxAssumptionLength caps the model's interpretation note. It is rendered
// to the user, so it is bounded.
const maxAssumptionLength = 400

// maxUnmappedItems caps how many unmapped clauses are reported back.
const maxUnmappedItems = 5

// maxExtraPlans caps how many extra catalogues one question may span.
const maxExtraPlans = 3

// Validate turns the model's DTO into a set of plans. Every field name,
func Validate(dto *FilterSetDTO) (*Plans, *ValidationError) {
	if dto == nil {
		return nil, &ValidationError{Message: "the model returned an empty filter set"}
	}

	primary, verr := validatePlan(dto.Entity, dto.Filters, dto.Related, dto.Sort, dto.Assumptions, dto.Unmapped)
	if verr != nil {
		return nil, verr
	}

	plans := &Plans{Primary: primary}

	seen := map[string]bool{primary.Entity.Name: true}
	for _, also := range dto.Also {
		if len(plans.Others) >= maxExtraPlans {
			break
		}
		// The same catalogue twice is one search, not two. Skipping it
		// keeps the results from showing the same row twice.
		if seen[also.Entity] {
			continue
		}
		other, verr := validatePlan(
			also.Entity, also.Filters, also.Related, also.Sort,
			also.Assumptions, nil,
		)
		if verr != nil {
			return nil, verr
		}
		seen[also.Entity] = true
		plans.Others = append(plans.Others, other)
	}

	return plans, nil
}

// validatePlan checks one catalogue's worth of filters and returns the plan
// they describe.
func validatePlan(
	entityName string,
	filters []ConditionDTO,
	related []RelatedDTO,
	sort *SortDTO,
	assumptions string,
	unmapped []string,
) (*Plan, *ValidationError) {
	entity, ok := entityByName(entityName)
	if !ok {
		return nil, &ValidationError{
			Field:   "entity",
			Message: fmt.Sprintf("unknown entity %q", entityName),
		}
	}

	plan := &Plan{
		Entity:      entity,
		Assumptions: truncate(strings.TrimSpace(assumptions), maxAssumptionLength),
		Unmapped:    trimAll(unmapped, maxUnmappedItems),
	}

	for _, c := range filters {
		spec, verr := validateCondition(entity, c)
		if verr != nil {
			return nil, verr
		}
		plan.Filters = append(plan.Filters, spec)
	}

	for _, rel := range related {
		relation, ok := entity.Related[rel.Entity]
		if !ok {
			return nil, &ValidationError{
				Field:   "related.entity",
				Message: fmt.Sprintf("%q has no related entity %q", entity.Name, rel.Entity),
			}
		}
		rs := RelatedSpec{Relation: relation}
		for _, c := range rel.Filters {
			spec, verr := validateCondition(relation.Entity, c)
			if verr != nil {
				return nil, verr
			}
			rs.Filters = append(rs.Filters, spec)
		}
		// A related group with no filters is still a constraint: it means
		// "only rows that have one of these", which the compiler writes as an
		// EXISTS with nothing but the join.
		plan.Related = append(plan.Related, rs)
	}

	if sort != nil {
		order, verr := validateSort(entity, sort)
		if verr != nil {
			return nil, verr
		}
		plan.Sort = order
	}

	return plan, nil
}

// validateCondition checks one constraint against a field definition.
func validateCondition(entity *Entity, c ConditionDTO) (FilterSpec, *ValidationError) {
	hasFilter := c.Field != ""
	hasGroup := len(c.Any) > 0

	switch {
	case hasFilter && hasGroup:
		return FilterSpec{}, &ValidationError{
			Field:   c.Field,
			Message: "a constraint is either one filter or a group of alternatives, not both",
		}
	case hasGroup:
		return validateGroup(entity, c.Any)
	case hasFilter:
		return validateFilter(entity, FilterDTO{
			Field: c.Field,
			Op:    c.Op,
			Value: c.Value,
		})
	default:
		return FilterSpec{}, &ValidationError{
			Message: "a constraint must name a field or list alternatives",
		}
	}
}

// validateGroup checks a set of alternatives.
func validateGroup(entity *Entity, alternatives []FilterDTO) (FilterSpec, *ValidationError) {
	if len(alternatives) == 0 {
		return FilterSpec{}, &ValidationError{
			Message: "a group of alternatives must list at least one",
		}
	}
	// A group of one is just a filter, so writing it as a group is probably
	// a mistake worth reporting.
	if len(alternatives) == 1 {
		return FilterSpec{}, &ValidationError{
			Field:   alternatives[0].Field,
			Message: "a group with one alternative is not a choice, write it as a plain filter",
		}
	}

	spec := FilterSpec{}
	for _, f := range alternatives {
		alt, verr := validateFilter(entity, f)
		if verr != nil {
			return FilterSpec{}, verr
		}
		spec.Any = append(spec.Any, alt)
	}
	return spec, nil
}

// validateFilter checks one constraint against a field definition and
// normalises its value.
func validateFilter(entity *Entity, f FilterDTO) (FilterSpec, *ValidationError) {
	field, ok := entity.Field(f.Field)
	if !ok {
		return FilterSpec{}, &ValidationError{
			Field:   f.Field,
			Message: fmt.Sprintf("unknown field for %s", entity.Name),
		}
	}

	op := Op(strings.TrimSpace(f.Op))
	if op == "" {
		op = OpEq
	}
	if !field.Allows(op) {
		return FilterSpec{}, &ValidationError{
			Field:   f.Field,
			Message: fmt.Sprintf("operator %q is not allowed on this field", op),
		}
	}

	value, verr := normaliseValue(field, op, f.Value)
	if verr != nil {
		return FilterSpec{}, verr
	}

	if len(field.Values) > 0 && op != OpNe {
		if err := checkClosedVocabulary(field, value); err != nil {
			return FilterSpec{}, err
		}
	}

	return FilterSpec{Field: field.Name, Op: op, Value: value}, nil
}

// validateSort checks the requested ordering.
func validateSort(entity *Entity, s *SortDTO) (*SortSpec, *ValidationError) {
	field, ok := entity.Field(s.Field)
	if !ok || !field.Sortable {
		return nil, &ValidationError{
			Field:   "sort.field",
			Message: fmt.Sprintf("cannot sort %s by %q", entity.Name, s.Field),
		}
	}
	dir := strings.ToLower(strings.TrimSpace(s.Direction))
	switch dir {
	case "asc", "":
		dir = "asc"
	case "desc":
	default:
		return nil, &ValidationError{
			Field:   "sort.direction",
			Message: fmt.Sprintf("unknown sort direction %q", s.Direction),
		}
	}
	return &SortSpec{Field: field.Name, Direction: dir}, nil
}

// normaliseValue coerces the model's value into the field's type. The model
func normaliseValue(field Field, op Op, raw any) (Value, *ValidationError) {
	invalid := func(reason string) *ValidationError {
		return &ValidationError{Field: field.Name, Message: reason}
	}

	switch field.Kind {
	case KindBool:
		switch v := raw.(type) {
		case bool:
			return Value{Kind: ValueBool, Bool: v}, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "true":
				return Value{Kind: ValueBool, Bool: true}, nil
			case "false":
				return Value{Kind: ValueBool, Bool: false}, nil
			}
		}
		return Value{}, invalid("expected a true/false value")

	case KindNumber:
		switch v := raw.(type) {
		case float64:
			return Value{Kind: ValueNumber, Num: v}, nil
		case json.Number:
			if f, err := v.Float64(); err == nil {
				return Value{Kind: ValueNumber, Num: f}, nil
			}
		case int:
			return Value{Kind: ValueNumber, Num: float64(v)}, nil
		case int64:
			return Value{Kind: ValueNumber, Num: float64(v)}, nil
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				return Value{Kind: ValueNumber, Num: f}, nil
			}
		}
		return Value{}, invalid("expected a number")

	case KindText, KindString:
		// A list is only meaningful for "in"; other operators take one value.
		if list, ok := raw.([]any); ok {
			if op != OpIn {
				return Value{}, invalid(fmt.Sprintf("operator %q takes a single value", op))
			}
			out := make([]string, 0, len(list))
			for _, item := range list {
				s, ok := item.(string)
				if !ok {
					return Value{}, invalid("expected a list of strings")
				}
				if s = strings.TrimSpace(s); s != "" {
					out = append(out, translateCode(field.Code, s))
				}
			}
			if len(out) == 0 {
				return Value{}, invalid("expected at least one value")
			}
			return Value{Kind: ValueList, List: out}, nil
		}
		switch v := raw.(type) {
		case string:
			if v = strings.TrimSpace(v); v != "" {
				return Value{Kind: ValueString, Str: translateCode(field.Code, v)}, nil
			}
		case float64:
			// Numbers are legitimate values for text columns ("founded in 1998").
			return Value{Kind: ValueString, Str: strconv.FormatFloat(v, 'f', -1, 64)}, nil
		}
		return Value{}, invalid("expected a text value")
	}

	return Value{}, invalid("unsupported field type")
}

// translateCode turns a country or state name into the code the column
// holds. A question says "the United States" where the catalogue says "US".
func translateCode(code Code, value string) string {
	switch code {
	case CodeCountry:
		return countryCode(value)
	case CodeState:
		return stateCode(value)
	}
	return value
}

// checkClosedVocabulary enforces a field's fixed value list.
func checkClosedVocabulary(field Field, value Value) *ValidationError {
	allowed := field.Values
	for _, candidate := range value.candidates() {
		match := false
		for _, a := range allowed {
			if strings.EqualFold(a, candidate) {
				match = true
				break
			}
		}
		if !match {
			return &ValidationError{
				Field:   field.Name,
				Message: fmt.Sprintf("%q is not one of: %s", candidate, strings.Join(allowed, ", ")),
			}
		}
	}
	return nil
}

// candidates returns the value(s) as strings, for vocabulary checks.
func (v Value) candidates() []string {
	switch v.Kind {
	case ValueList:
		return v.List
	case ValueString:
		return []string{v.Str}
	case ValueNumber:
		return []string{strconv.FormatFloat(v.Num, 'f', -1, 64)}
	case ValueBool:
		return []string{strconv.FormatBool(v.Bool)}
	}
	return nil
}

// truncate shortens a string to at most max characters, counting characters
// rather than bytes so a multi-byte rune is never cut in half.
func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func trimAll(items []string, max int) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, truncate(item, 120))
		if len(out) == max {
			break
		}
	}
	return out
}
