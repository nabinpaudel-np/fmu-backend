package nlsearch

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// The model speaks in two layers: a JSON DTO it produces, and a validated
// Plan the compiler consumes. Nothing between the two may be trusted — the
// DTO is re-read against the vocabulary here, and the compiler only ever
// sees values that came out of this file.

// FilterDTO is one constraint as the model writes it.
type FilterDTO struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// RelatedDTO is a set of constraints on a related entity.
type RelatedDTO struct {
	Entity  string      `json:"entity"`
	Filters []FilterDTO `json:"filters"`
}

// SortDTO is the requested ordering.
type SortDTO struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// FilterSetDTO is the model's entire output. It carries no prose the
// application acts on: `assumptions` and `unmapped` are shown to the user,
// never executed.
type FilterSetDTO struct {
	Entity      string       `json:"entity"`
	Filters     []FilterDTO  `json:"filters"`
	Related     []RelatedDTO `json:"related"`
	Sort        *SortDTO     `json:"sort"`
	Assumptions string       `json:"assumptions"`
	Unmapped    []string     `json:"unmapped"`
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

// Validate turns the model's DTO into a Plan. Every field name, operator and
// value is checked against the vocabulary: an unknown field is refused rather
// than ignored, because silently dropping it would return results that do
// not answer the question.
func Validate(dto *FilterSetDTO) (*Plan, *ValidationError) {
	if dto == nil {
		return nil, &ValidationError{Message: "the model returned an empty filter set"}
	}

	entity, ok := entityByName(dto.Entity)
	if !ok {
		return nil, &ValidationError{
			Field:   "entity",
			Message: fmt.Sprintf("unknown entity %q", dto.Entity),
		}
	}

	plan := &Plan{
		Entity:      entity,
		Assumptions: truncate(strings.TrimSpace(dto.Assumptions), maxAssumptionLength),
		Unmapped:    trimAll(dto.Unmapped, maxUnmappedItems),
	}

	for _, f := range dto.Filters {
		spec, verr := validateFilter(entity, f)
		if verr != nil {
			return nil, verr
		}
		plan.Filters = append(plan.Filters, spec)
	}

	for _, rel := range dto.Related {
		relation, ok := entity.Related[rel.Entity]
		if !ok {
			return nil, &ValidationError{
				Field:   "related.entity",
				Message: fmt.Sprintf("%q has no related entity %q", entity.Name, rel.Entity),
			}
		}
		rs := RelatedSpec{Relation: relation}
		for _, f := range rel.Filters {
			spec, verr := validateFilter(relation.Entity, f)
			if verr != nil {
				return nil, verr
			}
			rs.Filters = append(rs.Filters, spec)
		}
		if len(rs.Filters) > 0 {
			plan.Related = append(plan.Related, rs)
		}
	}

	if dto.Sort != nil {
		sort, verr := validateSort(entity, dto.Sort)
		if verr != nil {
			return nil, verr
		}
		plan.Sort = sort
	}

	return plan, nil
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
// is asked for typed JSON but is not trusted to comply, so numbers arriving
// as strings and booleans arriving as "true" are both accepted.
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
					out = append(out, s)
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
				return Value{Kind: ValueString, Str: v}, nil
			}
		case float64:
			// Numbers are legitimate values for text columns ("founded in 1998").
			return Value{Kind: ValueString, Str: strconv.FormatFloat(v, 'f', -1, 64)}, nil
		}
		return Value{}, invalid("expected a text value")
	}

	return Value{}, invalid("unsupported field type")
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
