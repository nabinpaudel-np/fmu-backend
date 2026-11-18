package countryalias

import "testing"

func TestCountryCode(t *testing.T) {
	cases := map[string]string{
		"US":                       "US",
		"us":                       "US",
		"usa":                      "US",
		"USA":                      "US",
		"United States of America": "US",
		"united states of america": "US",
		"america":                  "US",
		"united kingdom":           "GB",
		"uk":                       "GB",
		"Germany":                  "DE",
	}
	for input, want := range cases {
		if got := CountryCode(input); got != want {
			t.Errorf("CountryCode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMatchCountryNames(t *testing.T) {
	cases := []struct {
		query string
		want  bool // whether a country should be resolved
	}{
		{"US", true},
		{"USA", true},
		{"usa", true},
		{"United States of America", true},
		{"united states of america", true},
		{"America", true},
		{"universities in USA", true},
		{"universite in US", true},
		{"united kingdom", true},
		{"Canada", true},
		{"Harvard", false},
		{"universite", false},
		{"", false},
	}
	for _, tc := range cases {
		got := MatchCountryNames(tc.query)
		if (len(got) > 0) != tc.want {
			t.Errorf("MatchCountryNames(%q) = %v, want country resolved = %v", tc.query, got, tc.want)
		}
	}
}

func TestMatchCountryNamesExpandsUS(t *testing.T) {
	got := MatchCountryNames("USA")
	want := map[string]bool{
		"us":                       true,
		"usa":                      true,
		"america":                  true,
		"united states":            true,
		"united states of america": true,
	}
	for _, alias := range got {
		delete(want, alias)
	}
	if len(want) != 0 {
		t.Errorf("MatchCountryNames(\"USA\") missing aliases: %v (got %v)", want, got)
	}
}

func TestMatchCountryNamesIgnoresNonCountry(t *testing.T) {
	if got := MatchCountryNames("university of california"); got != nil {
		t.Errorf("MatchCountryNames(%q) = %v, want nil", "university of california", got)
	}
}

func TestStateCode(t *testing.T) {
	cases := map[string]string{
		"California": "CA",
		"california": "CA",
		"CA":         "CA",
		"New York":   "NY",
		"Texas":      "TX",
	}
	for input, want := range cases {
		if got := StateCode(input); got != want {
			t.Errorf("StateCode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMatchContinent(t *testing.T) {
	cases := []struct {
		query string
		want  string // expected catalogue spelling, "" for no match
	}{
		{"Europe", "Europe"},
		{"europe", "Europe"},
		{"North America", "North America"},
		{"north america", "North America"},
		{"universities in North America", "North America"},
		{"Asia", "Asia"},
		// "America" alone is NOT a continent — it is a country alias.
		// Matching it as a continent would pull every North American row
		// into a "United States of America" search.
		{"America", ""},
		{"United States of America", ""},
		{"Harvard", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := MatchContinent(tc.query)
		if tc.want == "" {
			if len(got) != 0 {
				t.Errorf("MatchContinent(%q) = %v, want no match", tc.query, got)
			}
			continue
		}
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("MatchContinent(%q) = %v, want [%q]", tc.query, got, tc.want)
		}
	}
}
