package nlsearch

import "fmu-backend/internal/countryalias"

func countryCode(value string) string {
	return countryalias.CountryCode(value)
}

func stateCode(value string) string {
	return countryalias.StateCode(value)
}
