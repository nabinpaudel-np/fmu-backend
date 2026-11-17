package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fmu-backend/internal/nlsearch"
)

// fakeAI reports a fixed set of keys.
type fakeAI struct {
	model string
	keys  []nlsearch.KeyStatus
}

func (f *fakeAI) KeyStatuses() []nlsearch.KeyStatus { return f.keys }
func (f *fakeAI) Model() string                     { return f.model }

func TestLiveness_AlwaysOk(t *testing.T) {
	h := NewHandler(nil, nil)

	rec := httptest.NewRecorder()
	h.Liveness(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "{\"status\":\"ok\"}\n" {
		t.Fatalf("got %q", got)
	}
}

func TestReadiness_OkWithoutAPool(t *testing.T) {
	h := NewHandler(nil, nil)

	rec := httptest.NewRecorder()
	h.Readiness(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestModel_NotConfigured(t *testing.T) {
	h := NewHandler(nil, nil)

	rec := httptest.NewRecorder()
	h.Model(rec, httptest.NewRequest(http.MethodGet, "/health/ai", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body AIHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "not configured" {
		t.Fatalf("got %q, want %q", body.Status, "not configured")
	}
}

func TestModel_Ok(t *testing.T) {
	h := NewHandler(nil, &fakeAI{
		model: "gemini-3.5-flash-lite",
		keys: []nlsearch.KeyStatus{
			{Index: 1, Available: true},
			{Index: 2, Available: true},
		},
	})

	rec := httptest.NewRecorder()
	h.Model(rec, httptest.NewRequest(http.MethodGet, "/health/ai", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}

	var body AIHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("got %q, want %q", body.Status, "ok")
	}
	if body.Model != "gemini-3.5-flash-lite" {
		t.Fatalf("got model %q", body.Model)
	}
	if body.KeysAvailable != 2 || body.KeysTotal != 2 {
		t.Fatalf("got %d of %d keys", body.KeysAvailable, body.KeysTotal)
	}
}

// One spent key is still a working setup.
func TestModel_PartlyAvailable(t *testing.T) {
	h := NewHandler(nil, &fakeAI{
		model: "m",
		keys: []nlsearch.KeyStatus{
			{Index: 1, Available: false, RetryAfterSeconds: 120},
			{Index: 2, Available: true},
		},
	})

	rec := httptest.NewRecorder()
	h.Model(rec, httptest.NewRequest(http.MethodGet, "/health/ai", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusOK)
	}

	var body AIHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.KeysAvailable != 1 || body.KeysTotal != 2 {
		t.Fatalf("got %d of %d keys", body.KeysAvailable, body.KeysTotal)
	}
}

// Every key spent means search will fail, so the endpoint says so.
func TestModel_NoKeyAvailable(t *testing.T) {
	h := NewHandler(nil, &fakeAI{
		model: "m",
		keys: []nlsearch.KeyStatus{
			{Index: 1, Available: false, RetryAfterSeconds: 60},
		},
	})

	rec := httptest.NewRecorder()
	h.Model(rec, httptest.NewRequest(http.MethodGet, "/health/ai", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body AIHealth
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "no key available" {
		t.Fatalf("got %q", body.Status)
	}
	if body.Message == "" {
		t.Fatal("expected a message explaining the state")
	}
}

// The key itself must never appear in the report.
func TestModel_DoesNotLeakTheKey(t *testing.T) {
	h := NewHandler(nil, &fakeAI{
		model: "m",
		keys:  []nlsearch.KeyStatus{{Index: 1, Available: true}},
	})

	rec := httptest.NewRecorder()
	h.Model(rec, httptest.NewRequest(http.MethodGet, "/health/ai", nil))

	if body := rec.Body.String(); strings.Contains(body, "AQ.Ab8") || strings.Contains(body, "API_KEY") {
		t.Fatalf("the response mentions the key: %s", body)
	}
}
