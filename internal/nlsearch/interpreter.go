package nlsearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/genai"
)

// The interpreter is the only part of the system that talks to a language
// model. Everything it returns is re-read against the vocabulary before it
// can affect a query.

var (
	// ErrProviderUnavailable means no model is configured, or the provider
	// rejected the key.
	ErrProviderUnavailable = errors.New("language model is not configured")
	// ErrProviderFailed means the model was reached but produced something
	// unusable. The caller fails closed.
	ErrProviderFailed = errors.New("language model returned an unusable answer")
	// ErrProviderQuotaExceeded means the provider's allowance for the day is
	// spent. The question was fine, so it is reported as an outage.
	ErrProviderQuotaExceeded = errors.New("language model quota is exhausted")
	// ErrProviderRateLimited means too many requests arrived in a short
	// window. It clears on its own.
	ErrProviderRateLimited = errors.New("language model rate limit reached")
	// ErrProviderBusy means the provider is saturated.
	ErrProviderBusy = errors.New("language model is busy")
)

// InterpretRequest is one question about the catalogue.
type InterpretRequest struct {
	Question   string
	Correction string
}

// Interpreter turns a question into a filter set.
type Interpreter interface {
	// Interpret turns a question into a filter set.
	Interpret(ctx context.Context, req InterpretRequest) (*FilterSetDTO, error)
}

// retryableError carries how long the caller should wait before trying again,
// alongside the sentinel that says what kind of failure it was.
type retryableError struct {
	err        error
	retryAfter time.Duration
}

func (e *retryableError) Error() string { return e.err.Error() }

func (e *retryableError) Unwrap() error { return e.err }

// GeminiInterpreter talks to the Gemini API.
type GeminiInterpreter struct {
	client *genai.Client
	model  string

	// slots bounds how many model calls are in flight at once across the
	// whole process. The provider's allowance is shared by everyone using the
	// key, so one client's burst must not spend it.
	slots semaphore

	// thinking records whether the model accepts a thinking configuration.
	// Some models reject the request when one is sent, and the provider's
	// allowance is counted in requests, so re-discovering that every call
	// would double what each search costs.
	thinking atomic.Int32
}

// modelSlots is how many model calls may run at once.
const modelSlots = 2

// semaphore counts the calls allowed to run at once. A request that cannot
// get a slot waits for one, or gives up when its context is done.
type semaphore chan struct{}

func newSemaphore(n int) semaphore {
	if n < 1 {
		n = 1
	}
	return make(semaphore, n)
}

// acquire takes a slot, waiting for one if none is free.
func (s semaphore) acquire(ctx context.Context) error {
	select {
	case s <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// release gives the slot back.
func (s semaphore) release() {
	<-s
}

const (
	// thinkingUnknown means the model has not refused a thinking
	// configuration yet, so one is still sent.
	thinkingUnknown = iota
	// thinkingRefused means the model rejected a request that carried one.
	thinkingRefused
)

// NewGeminiInterpreter builds an interpreter. An empty API key returns
// ErrProviderUnavailable: the app must still boot without a model.
func NewGeminiInterpreter(ctx context.Context, apiKey, model string) (*GeminiInterpreter, error) {
	return newGeminiInterpreter(ctx, apiKey, model, "")
}

// newGeminiInterpreter builds an interpreter against a given base URL. An
// empty base URL means the provider's own.
func newGeminiInterpreter(ctx context.Context, apiKey, model, baseURL string) (*GeminiInterpreter, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, ErrProviderUnavailable
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
		// The Gemini API reports a busy model as a 503 and a busy quota as a
		// 429. A busy model is momentary, so the request is retried once
		// rather than being reported to the user as a failure.
		//
		// The retryable codes are listed explicitly because the default set
		// includes 429, and the free tier's allowance is counted in requests
		// per day: retrying a 429 spends three more of the day's allowance
		// to reach the same answer. Only the codes worth waiting out are
		// retried, and only twice.
		HTTPOptions: genai.HTTPOptions{
			BaseURL: baseURL,
			RetryOptions: &genai.HTTPRetryOptions{
				Attempts:        genai.Ptr[int32](2),
				InitialDelay:    genai.Ptr[float64](1),
				MaxDelay:        genai.Ptr[float64](4),
				HTTPStatusCodes: []int32{500, 502, 503, 504},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &GeminiInterpreter{client: client, model: model, slots: newSemaphore(modelSlots)}, nil
}

// Interpret asks the model to translate the question into a filter set.
func (g *GeminiInterpreter) Interpret(ctx context.Context, req InterpretRequest) (*FilterSetDTO, error) {
	config := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		Temperature:       genai.Ptr[float32](0),
		ResponseMIMEType:  "application/json",
		ResponseSchema:    filterSetSchema(),
		// A question that spans several catalogues answers with a plan for
		// each, so the answer is longer than one plan's worth of filters.
		MaxOutputTokens: 4096,
		ThinkingConfig:  noThinking(),
	}

	prompt := buildUserPrompt(req.Question)
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
	return &dto, nil
}

// generate asks the model and returns the text it produced.
func (g *GeminiInterpreter) generate(ctx context.Context, prompt string, config *genai.GenerateContentConfig) (string, error) {
	if err := g.slots.acquire(ctx); err != nil {
		return "", err
	}
	defer g.slots.release()

	contents := []*genai.Content{genai.NewContentFromText(prompt, genai.RoleUser)}

	if g.thinking.Load() == thinkingRefused {
		config = withoutThinking(config)
	}

	resp, err := g.client.Models.GenerateContent(ctx, g.model, contents, config)
	if err != nil && isRefused(err) && config.ThinkingConfig != nil {
		// The model rejected the request itself, not the question. Some
		// models do not accept a thinking configuration at all, and asking
		// again without it is the only way to find out. The conclusion is
		// only believed when that second attempt succeeds: a 400 that
		// survives the retry was never about the thinking.
		plain := withoutThinking(config)
		resp, err = g.client.Models.GenerateContent(ctx, g.model, contents, plain)
		if err == nil {
			g.thinking.Store(thinkingRefused)
		}
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

// withoutThinking copies a configuration with the thinking removed.
func withoutThinking(config *genai.GenerateContentConfig) *genai.GenerateContentConfig {
	plain := *config
	plain.ThinkingConfig = nil
	return &plain
}

// isRefused reports whether the provider rejected the request itself, as
// opposed to failing to answer it.
func isRefused(err error) bool {
	var apiErr genai.APIError
	return errors.As(err, &apiErr) && apiErr.Code == http.StatusBadRequest
}

// providerError classifies a failed model call.
func providerError(err error) error {
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case http.StatusTooManyRequests:
			// The provider states how long to wait in the error's details. A
			// wait measured in hours is the day's allowance running out; a
			// wait measured in seconds is a burst limit, and it clears on its
			// own.
			if delay, ok := retryDelay(apiErr); ok && delay >= time.Hour {
				return &retryableError{
					err:        fmt.Errorf("%w: %s", ErrProviderQuotaExceeded, apiErr.Message),
					retryAfter: delay,
				}
			}
			return &retryableError{
				err:        fmt.Errorf("%w: %s", ErrProviderRateLimited, apiErr.Message),
				retryAfter: delayOrZero(apiErr),
			}
		case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout:
			return &retryableError{
				err:        fmt.Errorf("%w: %s", ErrProviderBusy, apiErr.Message),
				retryAfter: delayOrZero(apiErr),
			}
		case http.StatusUnauthorized, http.StatusForbidden, http.StatusPaymentRequired:
			return fmt.Errorf("%w: %s", ErrProviderUnavailable, apiErr.Message)
		}
	}
	return fmt.Errorf("%w: %v", ErrProviderFailed, err)
}

// retryDelay reads the wait the provider asked for out of an error's
// details. The provider states it as a duration ("22h48m0s").
func retryDelay(apiErr genai.APIError) (time.Duration, bool) {
	for _, detail := range apiErr.Details {
		raw, ok := detail["retryDelay"].(string)
		if !ok {
			continue
		}
		delay, err := time.ParseDuration(raw)
		if err != nil {
			continue
		}
		return delay, true
	}
	return 0, false
}

// delayOrZero is the provider's requested wait, or none.
func delayOrZero(apiErr genai.APIError) time.Duration {
	delay, ok := retryDelay(apiErr)
	if !ok {
		return 0
	}
	return delay
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

// filterSetSchema constrains the model's output.
func filterSetSchema() *genai.Schema {
	filterSchema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"field": {Type: genai.TypeString, Enum: allFieldNames()},
			"op":    {Type: genai.TypeString, Enum: allOperatorNames()},
			"value": {
				AnyOf: []*genai.Schema{
					{Type: genai.TypeString},
					{Type: genai.TypeNumber},
					{Type: genai.TypeBoolean},
					{Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
				},
			},
			// A choice: at least one of these must hold.
			"any": {
				Type:  genai.TypeArray,
				Items: plainFilterSchema(),
			},
		},
	}

	properties := map[string]*genai.Schema{
		"entity": {Type: genai.TypeString, Enum: allEntityNames()},
		// The free-text term matched against the catalogue's prose.
		"search": {Type: genai.TypeString},
		"filters": {
			Type:  genai.TypeArray,
			Items: filterSchema,
		},
		"sort": {
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"field":     {Type: genai.TypeString, Enum: allSortableFieldNames()},
				"direction": {Type: genai.TypeString, Enum: []string{"asc", "desc"}},
			},
			Required: []string{"field", "direction"},
		},
		"assumptions": {Type: genai.TypeString},
		"unmapped": {
			Type:  genai.TypeArray,
			Items: &genai.Schema{Type: genai.TypeString},
		},
		// The model may answer a question that spans several catalogues.
		"related": {
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"entity":  {Type: genai.TypeString, Enum: allEntityNames()},
					"filters": {Type: genai.TypeArray, Items: filterSchema},
				},
				Required: []string{"entity"},
			},
		},
		"also": {
			Type:  genai.TypeArray,
			Items: alsoSchema(),
		},
	}

	return &genai.Schema{
		Type:       genai.TypeObject,
		Properties: properties,
		Required:   []string{"entity", "filters"},
	}
}

// plainFilterSchema describes one constraint with no alternatives. It is
// what the members of a group are, so a group cannot nest another group.
func plainFilterSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"field": {Type: genai.TypeString, Enum: allFieldNames()},
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

// alsoSchema describes one extra plan over another catalogue.
func alsoSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"entity": {Type: genai.TypeString, Enum: allEntityNames()},
			"search": {Type: genai.TypeString},
			"filters": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"field": {Type: genai.TypeString, Enum: allFieldNames()},
						"op":    {Type: genai.TypeString, Enum: allOperatorNames()},
						"value": {
							AnyOf: []*genai.Schema{
								{Type: genai.TypeString},
								{Type: genai.TypeNumber},
								{Type: genai.TypeBoolean},
								{Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
							},
						},
						"any": {
							Type:  genai.TypeArray,
							Items: plainFilterSchema(),
						},
					},
				},
			},
			"sort": {
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"field":     {Type: genai.TypeString, Enum: allSortableFieldNames()},
					"direction": {Type: genai.TypeString, Enum: []string{"asc", "desc"}},
				},
				Required: []string{"field", "direction"},
			},
			"assumptions": {Type: genai.TypeString},
		},
		Required: []string{"entity"},
	}
}

// allEntityNames lists every catalogue's name.
func allEntityNames() []string {
	names := make([]string, 0, len(entities))
	for _, e := range entities {
		names = append(names, e.Name)
	}
	return names
}

// allFieldNames lists every field of every catalogue.
func allFieldNames() []string {
	seen := map[string]bool{}
	var names []string
	for _, e := range entities {
		for _, f := range e.Fields {
			if !seen[f.Name] {
				seen[f.Name] = true
				names = append(names, f.Name)
			}
		}
	}
	return names
}

// allSortableFieldNames lists every field any catalogue's results may be
// ordered by.
func allSortableFieldNames() []string {
	seen := map[string]bool{}
	var names []string
	for _, e := range entities {
		for _, f := range e.Fields {
			if f.Sortable && !seen[f.Name] {
				seen[f.Name] = true
				names = append(names, f.Name)
			}
		}
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
