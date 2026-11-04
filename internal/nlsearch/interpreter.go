package nlsearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	// ErrProviderQuotaExceeded means the provider's allowance for the day is
	// spent. The question was fine and no amount of rephrasing will help, so
	// it is reported as a temporary outage rather than as a bad question.
	ErrProviderQuotaExceeded = errors.New("language model quota is exhausted")
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
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
		// The Gemini API reports a busy model as a 503 and a busy quota as a
		// 429. A busy model is momentary, so the request is retried a few
		// times with a growing delay rather than being reported to the user
		// as a failure.
		//
		// The retryable codes are listed explicitly because the default set
		// includes 429, and the free tier's allowance is counted in requests
		// per day: retrying a 429 spends three more of the day's allowance
		// to reach the same answer. Only the codes worth waiting out are
		// retried.
		HTTPOptions: genai.HTTPOptions{
			RetryOptions: &genai.HTTPRetryOptions{
				Attempts:        genai.Ptr[int32](4),
				InitialDelay:    genai.Ptr[float64](1),
				MaxDelay:        genai.Ptr[float64](8),
				HTTPStatusCodes: []int32{500, 502, 503, 504},
			},
		},
	})
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
		ThinkingConfig:    noThinking(),
	}

	text, err := g.generate(ctx, buildClassifyPrompt(question), config)
	if err != nil {
		return nil, err
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
		ThinkingConfig:    noThinking(),
	}

	prompt := buildUserPrompt(entity, req.Question)
	if req.Correction != "" {
		prompt += "\n\nYour previous answer was rejected:\n" + req.Correction +
			"\nReturn a corrected filter object using only the fields and values listed above."
	}

	text, err := g.generate(ctx, prompt, config)
	if err != nil {
		return nil, err
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

// generate asks the model and returns the text it produced.
//
// These models reason before they answer, which costs about ten seconds a
// question and buys nothing here: the schema already says what the answer
// must look like, and the fields are chosen by the vocabulary rather than by
// judgement. The thinking is turned off for that reason.
//
// Not every model accepts a thinking configuration. One that refuses the
// request is asked again without it, so the model can be swapped without
// also having to be re-tuned.
func (g *GeminiInterpreter) generate(ctx context.Context, prompt string, config *genai.GenerateContentConfig) (string, error) {
	contents := []*genai.Content{genai.NewContentFromText(prompt, genai.RoleUser)}

	resp, err := g.client.Models.GenerateContent(ctx, g.model, contents, config)
	if err != nil && isRefused(err) {
		withoutThinking := *config
		withoutThinking.ThinkingConfig = nil
		resp, err = g.client.Models.GenerateContent(ctx, g.model, contents, &withoutThinking)
	}
	if err != nil {
		return "", providerError(err)
	}
	if len(resp.Candidates) == 0 {
		return "", ErrProviderFailed
	}

	candidate := resp.Candidates[0]
	if candidate.FinishReason != "" && candidate.FinishReason != genai.FinishReasonStop {
		return "", fmt.Errorf("%w: model stopped with %s", ErrProviderFailed, candidate.FinishReason)
	}

	text := candidateText(candidate)
	if text == "" {
		return "", ErrProviderFailed
	}
	return text, nil
}

// noThinking asks the model to answer directly.
func noThinking() *genai.ThinkingConfig {
	return &genai.ThinkingConfig{ThinkingBudget: genai.Ptr[int32](0)}
}

// isRefused reports whether the provider rejected the request itself, as
// opposed to failing to answer it.
func isRefused(err error) bool {
	var apiErr genai.APIError
	return errors.As(err, &apiErr) && apiErr.Code == http.StatusBadRequest
}

// providerError classifies a failed model call. A spent quota is worth
// telling apart from a model that would not answer: the question was fine,
// and asking it again in the user's own words will not change that.
func providerError(err error) error {
	var apiErr genai.APIError
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusTooManyRequests {
		return fmt.Errorf("%w: %s", ErrProviderQuotaExceeded, apiErr.Message)
	}
	return fmt.Errorf("%w: %v", ErrProviderFailed, err)
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
