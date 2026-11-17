package nlsearch

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"google.golang.org/genai"
)

func TestUsableKeys_DropsEmptyAndRepeated(t *testing.T) {
	got := UsableKeys([]string{"a", "", "  ", "b", "a", " c "})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestUsableKeys_EmptyWhenNothingSet(t *testing.T) {
	if got := UsableKeys([]string{"", "   "}); len(got) != 0 {
		t.Fatalf("got %v, want none", got)
	}
}

func TestIsKeyFailure(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"quota exceeded", ErrProviderQuotaExceeded, true},
		{"rate limited", ErrProviderRateLimited, true},
		{"busy", ErrProviderBusy, true},
		{"unavailable", ErrProviderUnavailable, true},
		{"failed", ErrProviderFailed, false},
		{"plain", errors.New("boom"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isKeyFailure(tc.err); got != tc.want {
				t.Fatalf("isKeyFailure(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestProvider_Exhaustion(t *testing.T) {
	p := &provider{}
	now := time.Now()

	if !p.available(now) {
		t.Fatal("a fresh key must be available")
	}

	p.exhaust(now.Add(time.Minute))
	if p.available(now) {
		t.Fatal("an exhausted key must not be available")
	}
	if !p.available(now.Add(2 * time.Minute)) {
		t.Fatal("an exhausted key must come back after its wait")
	}
}

func TestProvider_ExhaustWithNoWait(t *testing.T) {
	p := &provider{}
	p.exhaust(time.Time{})
	if p.available(time.Now()) {
		t.Fatal("an exhausted key must not be available")
	}
}

func TestExhaustUntil_UsesTheProvidersWait(t *testing.T) {
	err := &retryableError{
		err:        ErrProviderRateLimited,
		retryAfter: 45 * time.Second,
	}
	until := exhaustUntil(err)
	if until.IsZero() {
		t.Fatal("expected a wait")
	}
	if d := time.Until(until); d < 40*time.Second || d > 50*time.Second {
		t.Fatalf("wait is %v, want about 45s", d)
	}
}

func TestExhaustUntil_NoWaitForAPlainError(t *testing.T) {
	if got := exhaustUntil(ErrProviderFailed); !got.IsZero() {
		t.Fatalf("expected no wait, got %v", got)
	}
}

// A 429 from the provider is the case the extra keys exist for.
func TestProviderError_QuotaIsRetryable(t *testing.T) {
	apiErr := genai.APIError{
		Code:    http.StatusTooManyRequests,
		Message: "quota exceeded",
		Details: []map[string]any{{"retryDelay": "22h48m0s"}},
	}
	err := providerError(apiErr)

	if !errors.Is(err, ErrProviderQuotaExceeded) {
		t.Fatalf("got %v, want a quota error", err)
	}
	if !isKeyFailure(err) {
		t.Fatal("a quota error must move to the next key")
	}
}

func TestIsKeyRejection(t *testing.T) {
	cases := []struct {
		message string
		want    bool
	}{
		{"API key not valid. Please pass a valid API key.", true},
		{"API key invalid", true},
		{"The API key expired", true},
		{"Permission denied for this resource", true},
		{"Invalid argument: field", false},
		{"the question was too long", false},
	}
	for _, tc := range cases {
		t.Run(tc.message, func(t *testing.T) {
			if got := isKeyRejection(tc.message); got != tc.want {
				t.Fatalf("isKeyRejection(%q) = %v, want %v", tc.message, got, tc.want)
			}
		})
	}
}

// The provider reports a rejected key as a bad request, so it has to be
// recognised or no key is ever tried.
func TestProviderError_InvalidKeyIsAFailure(t *testing.T) {
	apiErr := genai.APIError{
		Code:    http.StatusBadRequest,
		Message: "API key not valid. Please pass a valid API key.",
	}
	err := providerError(apiErr)

	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("got %v, want the provider to be unavailable", err)
	}
	if !isKeyFailure(err) {
		t.Fatal("an invalid key must move to the next key")
	}
}

// A bad question must not be treated as a bad key.
func TestProviderError_BadQuestionIsNotAKeyFailure(t *testing.T) {
	apiErr := genai.APIError{
		Code:    http.StatusBadRequest,
		Message: "Invalid argument: contents",
	}
	err := providerError(apiErr)

	if isKeyFailure(err) {
		t.Fatal("a malformed question must not move to the next key")
	}
}
