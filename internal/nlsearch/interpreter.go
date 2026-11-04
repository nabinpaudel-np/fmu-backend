package nlsearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// The interpreter is the only part of the system that talks to a language
// model. Everything it returns is re-read against the vocabulary before it
// can affect a query, so the interface is deliberately narrow: a question in,
// a filter set out, and no way to express anything else.
//
// A question is answered in two steps. The first picks the catalogue the
// question is about. The second is given only that catalogue's fields, so
// the model cannot name a field that does not belong to it — the schema
// makes the wrong answer impossible rather than merely discouraged.

var (
	// ErrProviderUnavailable means no model is configured. The endpoint
	// reports that it is unavailable and the catalogue's own search
	// endpoints keep serving the frontend.
	ErrProviderUnavailable = errors.New("language model is not configured")
	// ErrProviderFailed means the model was reached but produced something
	// unusable. The caller fails closed.
	ErrProviderFailed = errors.New("language model returned an unusable answer")
)

// InterpretRequest is one question about a known catalogue. Correction
// carries the validation failure from a previous attempt so the model can fix
// itself once.
type InterpretRequest struct {
	Question string
	// Entity is the catalogue the question is about, as chosen by Classify.
	Entity string
	// Correction is the error from a rejected attempt, if any.
	Correction string
}

// Classification is the model's answer to "which catalogue is this about?".
type Classification struct {
	// Entity is the catalogue name.
	Entity string
	// Assumptions notes anything the model had to decide on the way.
	Assumptions string
}

// Interpreter turns a question into a filter set.
type Interpreter interface {
	// Classify picks the catalogue a question is about.
	Classify(ctx context.Context, question string) (*Classification, error)
	// Interpret turns a question about a known catalogue into a filter set.
	Interpret(ctx context.Context, req InterpretRequest) (*FilterSetDTO, error)
}

// GeminiInterpreter talks to the Gemini API.
type GeminiInterpreter struct {
	client *genai.Client
	model  string
}

// NewGeminiInterpreter builds an interpreter. An empty API key returns
// ErrProviderUnavailable: the app must still boot without a model.
func NewGeminiInterpreter(ctx context.Context, apiKey, model string) (*GeminiInterpreter, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, ErrProviderUnavailable
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &GeminiInterpreter{client: client, model: model}, nil
}

// Classify asks the model which catalogue the question is about.
func (g *GeminiInterpreter) Classify(ctx context.Context, question string) (*Classification, error) {
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(classifyPrompt, genai.RoleUser),
		Temperature:       genai.Ptr[float32](0),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    classifySchema(),
		MaxOutputTokens:   256,
	}

	resp, err := g.client.Models.GenerateContent(
		ctx,
		g.model,
		[]*genai.Content{genai.NewContentFromText(buildClassifyPrompt(question), genai.RoleUser)},
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderFailed, err)
	}
	if len(resp.Candidates) == 0 {
		return nil, ErrProviderFailed
	}

	candidate := resp.Candidates[0]
	if candidate.FinishReason != "" && candidate.FinishReason != genai.FinishReasonStop {
		return nil, fmt.Errorf("%w: model stopped with %s", ErrProviderFailed, candidate.FinishReason)
	}

	text := candidateText(candidate)
	if text == "" {
		return nil, ErrProviderFailed
	}

	var dto struct {
		Entity      string `json:"entity"`
		Assumptions string `json:"assumptions"`
	}
	if err := json.Unmarshal([]byte(text), &dto); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderFailed, err)
	}

	// The schema only allows the known names, so this should not happen.
	// Rather than fail the whole search, fall back to the default catalogue
	// and say so.
	if _, ok := entityByName(dto.Entity); !ok {
		dto.Entity = universityEntity.Name
		dto.Assumptions = strings.TrimSpace(dto.Assumptions + " could not tell which catalogue was meant, searched universities")
	}

	return &Classification{Entity: dto.Entity, Assumptions: dto.Assumptions}, nil
}

// Interpret asks the model to translate the question into a filter set.
func (g *GeminiInterpreter) Interpret(ctx context.Context, req InterpretRequest) (*FilterSetDTO, error) {
	entity, ok := entityByName(req.Entity)
	if !ok {
		return nil, fmt.Errorf("%w: unknown entity %q", ErrProviderFailed, req.Entity)
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		Temperature:       genai.Ptr[float32](0),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    filterSetSchema(entity),
		MaxOutputTokens:   2048,
	}

	prompt := buildUserPrompt(entity, req.Question)
	if req.Correction != "" {
		prompt += "\n\nYour previous answer was rejected:\n" + req.Correction +
			"\nReturn a corrected filter object using only the fields and values listed above."
	}

	resp, err := g.client.Models.GenerateContent(
		ctx,
		g.model,
		[]*genai.Content{genai.NewContentFromText(prompt, genai.RoleUser)},
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderFailed, err)
	}
	if len(resp.Candidates) == 0 {
		return nil, ErrProviderFailed
	}

	candidate := resp.Candidates[0]
	if candidate.FinishReason != "" && candidate.FinishReason != genai.FinishReasonStop {
		return nil, fmt.Errorf("%w: model stopped with %s", ErrProviderFailed, candidate.FinishReason)
	}

	text := candidateText(candidate)
	if text == "" {
		return nil, ErrProviderFailed
	}

	var dto FilterSetDTO
	if err := json.Unmarshal([]byte(text), &dto); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderFailed, err)
	}
	// The catalogue was decided before the model was asked, so it is set
	// here rather than read from the answer.
	dto.Entity = entity.Name
	return &dto, nil
}

// candidateText joins the text parts of a candidate.
func candidateText(candidate *genai.Candidate) string {
	if candidate.Content == nil {
		return ""
	}
	var b strings.Builder
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			b.WriteString(part.Text)
		}
	}
	return strings.TrimSpace(b.String())
}

// classifySchema constrains the catalogue choice to the known names.
func classifySchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"entity": {Type: genai.TypeString, Enum: entityNames()},
		},
		Required: []string{"entity"},
	}
}

// filterSetSchema constrains the model's output to one catalogue's
// vocabulary. Field names and operators are enums, so the model cannot emit
// a field the server would have to reject; values stay free-form because the
// server checks them against the data.
func filterSetSchema(entity *Entity) *genai.Schema {
	filterSchema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"field": {Type: genai.TypeString, Enum: entity.FieldNames()},
			"op":    {Type: genai.TypeString, Enum: allOperatorNames()},
			"value": {
				AnyOf: []*genai.Schema{
					{Type: genai.TypeString},
					{Type: genai.TypeNumber},
					{Type: genai.TypeBoolean},
					{Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
				},
			},
		},
		Required: []string{"field", "op", "value"},
	}

	properties := map[string]*genai.Schema{
		"filters": {
			Type:  genai.TypeArray,
			Items: filterSchema,
		},
		"sort": {
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"field":     {Type: genai.TypeString, Enum: sortableFieldNames(entity)},
				"direction": {Type: genai.TypeString, Enum: []string{"asc", "desc"}},
			},
			Required: []string{"field", "direction"},
		},
		"assumptions": {Type: genai.TypeString},
		"unmapped": {
			Type:  genai.TypeArray,
			Items: &genai.Schema{Type: genai.TypeString},
		},
	}

	if len(entity.Related) > 0 {
		properties["related"] = &genai.Schema{
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"entity":  {Type: genai.TypeString, Enum: sortedRelatedNames(entity)},
					"filters": {Type: genai.TypeArray, Items: relatedFilterSchema(entity)},
				},
				Required: []string{"entity", "filters"},
			},
		}
	}

	return &genai.Schema{
		Type:       genai.TypeObject,
		Properties: properties,
		Required:   []string{"filters"},
	}
}

// relatedFilterSchema constrains a filter on a related catalogue to the
// fields the related catalogues own. Each entry in "related" names its own
// catalogue, and the validator rejects a field that belongs to a different
// one.
func relatedFilterSchema(entity *Entity) *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"field": {Type: genai.TypeString, Enum: relatedFieldNames(entity)},
			"op":    {Type: genai.TypeString, Enum: allOperatorNames()},
			"value": {
				AnyOf: []*genai.Schema{
					{Type: genai.TypeString},
					{Type: genai.TypeNumber},
					{Type: genai.TypeBoolean},
					{Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
				},
			},
		},
		Required: []string{"field", "op", "value"},
	}
}

// relatedFieldNames lists the fields of every catalogue reachable from this
// one.
func relatedFieldNames(entity *Entity) []string {
	var names []string
	seen := map[string]bool{}
	for _, name := range sortedRelatedNames(entity) {
		for _, field := range entity.Related[name].Entity.Fields {
			if !seen[field.Name] {
				seen[field.Name] = true
				names = append(names, field.Name)
			}
		}
	}
	return names
}

func entityNames() []string {
	names := make([]string, 0, len(entities))
	for _, e := range entities {
		names = append(names, e.Name)
	}
	return names
}

// sortableFieldNames lists the fields a catalogue's results may be ordered
// by.
func sortableFieldNames(entity *Entity) []string {
	names := make([]string, 0, len(entity.Fields))
	for _, f := range entity.Fields {
		if f.Sortable {
			names = append(names, f.Name)
		}
	}
	return names
}

func allOperatorNames() []string {
	return []string{"=", "!=", ">", ">=", "<", "<=", "in", "contains", "matches"}
}
