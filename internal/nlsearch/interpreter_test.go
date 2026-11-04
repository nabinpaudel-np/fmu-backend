package nlsearch

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/genai"
)

// A spent allowance and a model that would not answer are different
// problems, and the user is told something different about each. The first
// is found by the status the provider reported.
func TestProviderError_TellsASpentQuotaApartFromAFailedAnswer(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "the daily allowance is spent",
			err:  genai.APIError{Code: 429, Message: "quota exceeded", Status: "RESOURCE_EXHAUSTED"},
			want: ErrProviderQuotaExceeded,
		},
		{
			name: "the allowance is spent and the provider wrapped the error",
			err:  fmt.Errorf("doRequest: error sending request: %w", genai.APIError{Code: 429}),
			want: ErrProviderQuotaExceeded,
		},
		{
			name: "the provider is having a bad time",
			err:  genai.APIError{Code: 503, Message: "model is overloaded"},
			want: ErrProviderFailed,
		},
		{
			name: "the request never left",
			err:  errors.New("connection refused"),
			want: ErrProviderFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := providerError(tc.err)
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

// The provider's complaint is kept in the error so it can be logged, but the
// error must still name the case it belongs to.
func TestProviderError_KeepsTheProvidersComplaint(t *testing.T) {
	err := providerError(genai.APIError{Code: 429, Message: "limit: 20 per day"})

	if !errors.Is(err, ErrProviderQuotaExceeded) {
		t.Fatalf("expected a quota error, got %v", err)
	}
	if got := err.Error(); !strings.Contains(got, "limit: 20 per day") {
		t.Errorf("expected the provider's message to survive, got %q", got)
	}
}
