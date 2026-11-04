package nlsearch

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestHandler(interpreter Interpreter, repo Repository) *Handler {
	return NewHandler(NewService(interpreter, repo))
}

func postSearch(t *testing.T, h *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nl-search", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.Search(rec, req)
	return rec
}

func TestHandler_SearchReturnsTheResults(t *testing.T) {
	interpreter := &fakeInterpreter{answers: []*FilterSetDTO{{
		Entity:  "universities",
		Filters: []FilterDTO{{Field: "city", Op: "=", Value: "Kathmandu"}},
	}}}
	repo := &fakeRepository{
		items: []ResultItem{{Entity: "universities", ID: "u1", Title: "Kathmandu University"}},
		total: 1,
	}
	rec := postSearch(t, newTestHandler(interpreter, repo), `{"query":"universities in kathmandu"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Query          string         `json:"query"`
			Interpretation Interpretation `json:"interpretation"`
			Items          []ResultItem   `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !envelope.Success {
		t.Error("expected the envelope to report success")
	}
	if envelope.Data.Query != "universities in kathmandu" {
		t.Errorf("expected the question to be echoed, got %q", envelope.Data.Query)
	}
	if len(envelope.Data.Items) != 1 || envelope.Data.Items[0].Title != "Kathmandu University" {
		t.Errorf("expected the result rows, got %#v", envelope.Data.Items)
	}
}

func TestHandler_SearchStatusCodes(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		interpreter *fakeInterpreter
		repo        *fakeRepository
		wantStatus  int
	}{
		{
			name:        "a body that is not JSON",
			body:        `not json`,
			interpreter: &fakeInterpreter{},
			repo:        &fakeRepository{},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "no question",
			body:        `{"query":""}`,
			interpreter: &fakeInterpreter{},
			repo:        &fakeRepository{},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name: "a filter the model got wrong",
			body: `{"query":"universities with good vibes"}`,
			interpreter: &fakeInterpreter{answers: []*FilterSetDTO{
				{Entity: "universities", Filters: []FilterDTO{{Field: "vibes", Op: "=", Value: "good"}}},
				{Entity: "universities", Filters: []FilterDTO{{Field: "vibes", Op: "=", Value: "good"}}},
			}},
			repo:       &fakeRepository{},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:        "the model is not configured",
			body:        `{"query":"universities"}`,
			interpreter: &fakeInterpreter{errs: []error{ErrProviderUnavailable}},
			repo:        &fakeRepository{},
			wantStatus:  http.StatusServiceUnavailable,
		},
		{
			name:        "the model could not be reached",
			body:        `{"query":"universities"}`,
			interpreter: &fakeInterpreter{errs: []error{ErrProviderFailed}},
			repo:        &fakeRepository{},
			wantStatus:  http.StatusBadGateway,
		},
		{
			// The question was fine; the server has spent its allowance.
			// Sending the user off to rephrase would not help.
			name:        "the model's allowance for the day is spent",
			body:        `{"query":"universities"}`,
			interpreter: &fakeInterpreter{errs: []error{ErrProviderQuotaExceeded}},
			repo:        &fakeRepository{},
			wantStatus:  http.StatusServiceUnavailable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := postSearch(t, newTestHandler(tc.interpreter, tc.repo), tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRateLimiter_LetsABurstThroughAndThenRefuses(t *testing.T) {
	limiter := NewRateLimiter(60, 3)

	for i := range 3 {
		if !limiter.Allow("ip:1.2.3.4") {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}
	if limiter.Allow("ip:1.2.3.4") {
		t.Fatal("expected the fourth request to be refused")
	}
	if !limiter.Allow("ip:5.6.7.8") {
		t.Fatal("expected a different caller to have its own allowance")
	}
}

func TestRateLimiter_RefusesWithTooManyRequests(t *testing.T) {
	limiter := NewRateLimiter(60, 1)
	handler := limiter.Limit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/nl-search", nil)
	req.RemoteAddr = "1.2.3.4:5000"

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, req)
	if first.Code != http.StatusOK {
		t.Fatalf("expected the first request to pass, got %d", first.Code)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Error("expected a Retry-After header so the client knows when to come back")
	}
}
