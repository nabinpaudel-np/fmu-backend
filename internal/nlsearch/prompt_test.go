package nlsearch

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// The prompt and the validator are two views of the same vocabulary. A field
// the model can see but the validator would refuse makes the model look
// broken; a field the validator accepts but the model cannot see makes a
// question impossible to answer. These tests keep the two views identical.

func TestRenderVocabulary_CoversTheCatalogueItIsGiven(t *testing.T) {
	for _, entity := range entities {
		vocabulary := renderVocabulary(entity)

		if !strings.Contains(vocabulary, entity.Name) {
			t.Errorf("the vocabulary does not mention the entity %q", entity.Name)
		}
		for _, field := range entity.Fields {
			if !strings.Contains(vocabulary, field.Name) {
				t.Errorf("the vocabulary does not mention the field %q of %q", field.Name, entity.Name)
			}
			if !strings.Contains(vocabulary, field.Label) {
				t.Errorf("the vocabulary does not describe the field %q", field.Name)
			}
			for _, value := range field.Values {
				if !strings.Contains(vocabulary, value) {
					t.Errorf("the vocabulary does not list the value %q of %q", value, field.Name)
				}
			}
		}
		for name, relation := range entity.Related {
			if !strings.Contains(vocabulary, name) {
				t.Errorf("the vocabulary does not mention the related entity %q", name)
			}
			for _, field := range relation.Entity.Fields {
				if !strings.Contains(vocabulary, field.Name) {
					t.Errorf("the vocabulary does not mention the related field %q", field.Name)
				}
			}
		}
	}
}

func TestRenderVocabulary_HidesTheOtherCatalogues(t *testing.T) {
	// A college question must not offer the model university fields: the
	// schema is per catalogue so the wrong field cannot be named at all.
	vocabulary := renderVocabulary(collegeEntity)

	if strings.Contains(vocabulary, "tuition_min") {
		t.Error("the college vocabulary mentions a university field")
	}
	if !strings.Contains(vocabulary, "campus_size") {
		t.Error("the college vocabulary does not mention a college field")
	}
}

func TestBuildUserPrompt_WrapsTheQuestionAsData(t *testing.T) {
	prompt := buildUserPrompt(universityEntity, "universities in kathmandu")

	if !strings.Contains(prompt, "<request>") || !strings.Contains(prompt, "</request>") {
		t.Fatalf("expected the question to be delimited, got %q", prompt)
	}
	if !strings.Contains(prompt, "universities in kathmandu") {
		t.Fatalf("expected the question to be included, got %q", prompt)
	}
	if !strings.Contains(systemPrompt, "never instructions") {
		t.Error("expected the system prompt to say the request is data, not instructions")
	}
}

func TestBuildClassifyPrompt_ListsEveryCatalogue(t *testing.T) {
	prompt := buildClassifyPrompt("scholarships for engineering students")

	for _, entity := range entities {
		if !strings.Contains(prompt, entity.Description) {
			t.Errorf("the classify prompt does not describe %q", entity.Name)
		}
	}
	if !strings.Contains(prompt, "<request>") {
		t.Error("expected the question to be delimited")
	}
}

func TestClassifySchema_AllowsOnlyKnownCatalogues(t *testing.T) {
	schema := classifySchema()

	names := schema.Properties["entity"].Enum
	if len(names) != len(entityNames()) {
		t.Errorf("expected %d catalogues in the schema, got %d", len(entityNames()), len(names))
	}
	for _, name := range entityNames() {
		if !slices.Contains(names, name) {
			t.Errorf("the schema does not allow the catalogue %q", name)
		}
	}
}

func TestFilterSetSchema_AllowsOnlyThatCataloguesFields(t *testing.T) {
	for _, entity := range entities {
		schema := filterSetSchema(entity)
		filterSchema := schema.Properties["filters"].Items

		allowed := filterSchema.Properties["field"].Enum
		if len(allowed) != len(entity.Fields) {
			t.Errorf("%s: expected %d fields in the schema, got %d", entity.Name, len(entity.Fields), len(allowed))
		}
		for _, field := range entity.Fields {
			if !slices.Contains(allowed, field.Name) {
				t.Errorf("%s: the schema does not allow the field %q", entity.Name, field.Name)
			}
		}
		for _, op := range allOperatorNames() {
			if !slices.Contains(filterSchema.Properties["op"].Enum, op) {
				t.Errorf("%s: the schema does not allow the operator %q", entity.Name, op)
			}
		}
	}
}

// TestEverySchemaFieldIsValidatable is the invariant that matters: whatever
// the schema lets the model name, the validator must accept.
func TestEverySchemaFieldIsValidatable(t *testing.T) {
	for _, entity := range entities {
		schema := filterSetSchema(entity)
		filterSchema := schema.Properties["filters"].Items

		for _, fieldName := range filterSchema.Properties["field"].Enum {
			field, ok := entity.Field(fieldName)
			if !ok {
				t.Fatalf("%s: the schema names the unknown field %q", entity.Name, fieldName)
			}
			for _, op := range filterSchema.Properties["op"].Enum {
				if !field.Allows(Op(op)) {
					continue
				}
				value := sampleValue(field, Op(op))
				if value == nil {
					continue
				}
				dto := &FilterSetDTO{
					Entity:  entity.Name,
					Filters: []FilterDTO{{Field: fieldName, Op: op, Value: value}},
				}
				if _, verr := Validate(dto); verr != nil {
					t.Errorf("the schema allows %s.%s %s %v but the validator refused it: %v",
						entity.Name, fieldName, op, value, verr)
				}
			}
		}
	}
}

// TestEveryRelatedFieldIsValidatable holds the same invariant for the
// filters that reach into a related catalogue.
func TestEveryRelatedFieldIsValidatable(t *testing.T) {
	for _, entity := range entities {
		for name, relation := range entity.Related {
			for _, field := range relation.Entity.Fields {
				for _, op := range field.Ops {
					value := sampleValue(field, op)
					if value == nil {
						continue
					}
					dto := &FilterSetDTO{
						Entity: entity.Name,
						Related: []RelatedDTO{{
							Entity:  name,
							Filters: []FilterDTO{{Field: field.Name, Op: string(op), Value: value}},
						}},
					}
					if _, verr := Validate(dto); verr != nil {
						t.Errorf("the vocabulary allows %s -> %s.%s %s %v but the validator refused it: %v",
							entity.Name, name, field.Name, op, value, verr)
					}
				}
			}
		}
	}
}

// sampleValue picks a value the validator should accept for a field and
// operator, or nil when the combination has no sensible value.
func sampleValue(field Field, op Op) any {
	switch field.Kind {
	case KindBool:
		if op != OpEq && op != OpNe {
			return nil
		}
		return true
	case KindNumber:
		switch op {
		case OpEq, OpNe, OpGt, OpGte, OpLt, OpLte:
			return 10
		}
		return nil
	default:
		switch op {
		case OpEq, OpNe, OpContains, OpMatches:
			if len(field.Values) > 0 {
				return field.Values[0]
			}
			return "Kathmandu"
		case OpIn:
			if len(field.Values) > 0 {
				return []any{field.Values[0]}
			}
			return []any{"Nepal"}
		}
		return nil
	}
}

func TestNewGeminiInterpreter_WithoutAKey(t *testing.T) {
	_, err := NewGeminiInterpreter(context.Background(), "  ", "gemini-3.5-flash")
	if err != ErrProviderUnavailable {
		t.Fatalf("expected the provider to be reported as unavailable, got %v", err)
	}
}
