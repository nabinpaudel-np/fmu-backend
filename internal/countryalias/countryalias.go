package countryalias

import (
	"sort"
	"strings"
)

var countryCodes = map[string]string{
	"afghanistan":                      "AF",
	"aland islands":                    "AX",
	"albania":                          "AL",
	"algeria":                          "DZ",
	"american samoa":                   "AS",
	"andorra":                          "AD",
	"angola":                           "AO",
	"anguilla":                         "AI",
	"antarctica":                       "AQ",
	"antigua and barbuda":              "AG",
	"argentina":                        "AR",
	"armenia":                          "AM",
	"aruba":                            "AW",
	"australia":                        "AU",
	"austria":                          "AT",
	"azerbaijan":                       "AZ",
	"bahamas":                          "BS",
	"bahrain":                          "BH",
	"bangladesh":                       "BD",
	"barbados":                         "BB",
	"belarus":                          "BY",
	"belgium":                          "BE",
	"belize":                           "BZ",
	"benin":                            "BJ",
	"bermuda":                          "BM",
	"bhutan":                           "BT",
	"bolivia":                          "BO",
	"bonaire":                          "BQ",
	"bosnia and herzegovina":           "BA",
	"botswana":                         "BW",
	"bouvet island":                    "BV",
	"brazil":                           "BR",
	"british indian ocean territory":   "IO",
	"brunei":                           "BN",
	"bulgaria":                         "BG",
	"burkina faso":                     "BF",
	"burundi":                          "BI",
	"cambodia":                         "KH",
	"cameroon":                         "CM",
	"canada":                           "CA",
	"cape verde":                       "CV",
	"cayman islands":                   "KY",
	"central african republic":         "CF",
	"chad":                             "TD",
	"chile":                            "CL",
	"china":                            "CN",
	"christmas island":                 "CX",
	"cocos islands":                    "CC",
	"colombia":                         "CO",
	"comoros":                          "KM",
	"congo":                            "CG",
	"cook islands":                     "CK",
	"costa rica":                       "CR",
	"cote d'ivoire":                    "CI",
	"croatia":                          "HR",
	"cuba":                             "CU",
	"curacao":                          "CW",
	"cyprus":                           "CY",
	"czech republic":                   "CZ",
	"czechia":                          "CZ",
	"democratic republic of the congo": "CD",
	"denmark":                          "DK",
	"djibouti":                         "DJ",
	"dominica":                         "DM",
	"dominican republic":               "DO",
	"ecuador":                          "EC",
	"egypt":                            "EG",
	"el salvador":                      "SV",
	"equatorial guinea":                "GQ",
	"eritrea":                          "ER",
	"estonia":                          "EE",
	"eswatini":                         "SZ",
	"ethiopia":                         "ET",
	"falkland islands":                 "FK",
	"faroe islands":                    "FO",
	"fiji":                             "FJ",
	"finland":                          "FI",
	"france":                           "FR",
	"french guiana":                    "GF",
	"french polynesia":                 "PF",
	"french southern territories":      "TF",
	"gabon":                            "GA",
	"gambia":                           "GM",
	"georgia":                          "GE",
	"germany":                          "DE",
	"ghana":                            "GH",
	"gibraltar":                        "GI",
	"greece":                           "GR",
	"greenland":                        "GL",
	"grenada":                          "GD",
	"guadeloupe":                       "GP",
	"guam":                             "GU",
	"guatemala":                        "GT",
	"guernsey":                         "GG",
	"guinea":                           "GN",
	"guinea-bissau":                    "GW",
	"guyana":                           "GY",
	"haiti":                            "HT",
	"heard island":                     "HM",
	"holy see":                         "VA",
	"honduras":                         "HN",
	"hong kong":                        "HK",
	"hungary":                          "HU",
	"iceland":                          "IS",
	"india":                            "IN",
	"indonesia":                        "ID",
	"iran":                             "IR",
	"iraq":                             "IQ",
	"ireland":                          "IE",
	"isle of man":                      "IM",
	"israel":                           "IL",
	"italy":                            "IT",
	"jamaica":                          "JM",
	"japan":                            "JP",
	"jersey":                           "JE",
	"jordan":                           "JO",
	"kazakhstan":                       "KZ",
	"kenya":                            "KE",
	"kiribati":                         "KI",
	"korea, north":                     "KP",
	"korea, south":                     "KR",
	"kosovo":                           "XK",
	"kuwait":                           "KW",
	"kyrgyzstan":                       "KG",
	"laos":                             "LA",
	"latvia":                           "LV",
	"lebanon":                          "LB",
	"lesotho":                          "LS",
	"liberia":                          "LR",
	"libya":                            "LY",
	"liechtenstein":                    "LI",
	"lithuania":                        "LT",
	"luxembourg":                       "LU",
	"macao":                            "MO",
	"madagascar":                       "MG",
	"malawi":                           "MW",
	"malaysia":                         "MY",
	"maldives":                         "MV",
	"mali":                             "ML",
	"malta":                            "MT",
	"marshall islands":                 "MH",
	"martinique":                       "MQ",
	"mauritania":                       "MR",
	"mauritius":                        "MU",
	"mayotte":                          "YT",
	"mexico":                           "MX",
	"micronesia":                       "FM",
	"moldova":                          "MD",
	"monaco":                           "MC",
	"mongolia":                         "MN",
	"montenegro":                       "ME",
	"montserrat":                       "MS",
	"morocco":                          "MA",
	"mozambique":                       "MZ",
	"myanmar":                          "MM",
	"namibia":                          "NA",
	"nauru":                            "NR",
	"nepal":                            "NP",
	"netherlands":                      "NL",
	"new caledonia":                    "NC",
	"new zealand":                      "NZ",
	"nicaragua":                        "NI",
	"niger":                            "NE",
	"nigeria":                          "NG",
	"niue":                             "NU",
	"norfolk island":                   "NF",
	"north korea":                      "KP",
	"north macedonia":                  "MK",
	"northern mariana islands":         "MP",
	"norway":                           "NO",
	"oman":                             "OM",
	"pakistan":                         "PK",
	"palau":                            "PW",
	"palestine":                        "PS",
	"panama":                           "PA",
	"papua new guinea":                 "PG",
	"paraguay":                         "PY",
	"peru":                             "PE",
	"philippines":                      "PH",
	"pitcairn":                         "PN",
	"poland":                           "PL",
	"portugal":                         "PT",
	"puerto rico":                      "PR",
	"qatar":                            "QA",
	"reunion":                          "RE",
	"romania":                          "RO",
	"russia":                           "RU",
	"rwanda":                           "RW",
	"saint barthelemy":                 "BL",
	"saint helena":                     "SH",
	"saint kitts and nevis":            "KN",
	"saint lucia":                      "LC",
	"saint martin":                     "MF",
	"saint pierre and miquelon":        "PM",
	"saint vincent and the grenadines": "VC",
	"samoa":                            "WS",
	"san marino":                       "SM",
	"sao tome and principe":            "ST",
	"saudi arabia":                     "SA",
	"senegal":                          "SN",
	"serbia":                           "RS",
	"seychelles":                       "SC",
	"sierra leone":                     "SL",
	"singapore":                        "SG",
	"sint maarten":                     "SX",
	"slovakia":                         "SK",
	"slovenia":                         "SI",
	"solomon islands":                  "SB",
	"somalia":                          "SO",
	"south africa":                     "ZA",
	"south korea":                      "KR",
	"south sudan":                      "SS",
	"spain":                            "ES",
	"sri lanka":                        "LK",
	"sudan":                            "SD",
	"suriname":                         "SR",
	"svalbard and jan mayen":           "SJ",
	"sweden":                           "SE",
	"switzerland":                      "CH",
	"syria":                            "SY",
	"taiwan":                           "TW",
	"tajikistan":                       "TJ",
	"tanzania":                         "TZ",
	"thailand":                         "TH",
	"timor-leste":                      "TL",
	"togo":                             "TG",
	"tokelau":                          "TK",
	"tonga":                            "TO",
	"trinidad and tobago":              "TT",
	"tunisia":                          "TN",
	"turkey":                           "TR",
	"turkmenistan":                     "TM",
	"turks and caicos islands":         "TC",
	"tuvalu":                           "TV",
	"uganda":                           "UG",
	"ukraine":                          "UA",
	"united arab emirates":             "AE",
	"united kingdom":                   "GB",
	"united states":                    "US",
	"united states of america":         "US",
	"uruguay":                          "UY",
	"uzbekistan":                       "UZ",
	"vanuatu":                          "VU",
	"vatican city":                     "VA",
	"venezuela":                        "VE",
	"vietnam":                          "VN",
	"wallis and futuna":                "WF",
	"western sahara":                   "EH",
	"yemen":                            "YE",
	"zambia":                           "ZM",
	"zimbabwe":                         "ZW",

	"america":               "US",
	"britain":               "GB",
	"burma":                 "MM",
	"cabo verde":            "CV",
	"cap verde":             "CV",
	"congo kinshasa":        "CD",
	"congo brazzaville":     "CG",
	"dr congo":              "CD",
	"drc":                   "CD",
	"england":               "GB",
	"great britain":         "GB",
	"holland":               "NL",
	"hong kong sar":         "HK",
	"ivory coast":           "CI",
	"macedonia":             "MK",
	"northern ireland":      "GB",
	"republic of ireland":   "IE",
	"republic of the congo": "CG",
	"scotland":              "GB",
	"swaziland":             "SZ",
	"the bahamas":           "BS",
	"the gambia":            "GM",
	"the netherlands":       "NL",
	"timor leste":           "TL",
	"turkiye":               "TR",
	"uae":                   "AE",
	"uk":                    "GB",
	"usa":                   "US",
	"wales":                 "GB",
}

func init() {
	for _, code := range countryCodes {
		lowered := strings.ToLower(code)
		if _, exists := countryCodes[lowered]; !exists {
			countryCodes[lowered] = code
		}
	}
}

var stateCodes = map[string]string{
	"alabama":                      "AL",
	"alaska":                       "AK",
	"american samoa":               "AS",
	"arizona":                      "AZ",
	"arkansas":                     "AR",
	"california":                   "CA",
	"colorado":                     "CO",
	"connecticut":                  "CT",
	"delaware":                     "DE",
	"district of columbia":         "DC",
	"florida":                      "FL",
	"georgia":                      "GA",
	"guam":                         "GU",
	"hawaii":                       "HI",
	"idaho":                        "ID",
	"illinois":                     "IL",
	"indiana":                      "IN",
	"iowa":                         "IA",
	"kansas":                       "KS",
	"kentucky":                     "KY",
	"louisiana":                    "LA",
	"maine":                        "ME",
	"maryland":                     "MD",
	"massachusetts":                "MA",
	"michigan":                     "MI",
	"minnesota":                    "MN",
	"mississippi":                  "MS",
	"missouri":                     "MO",
	"montana":                      "MT",
	"nebraska":                     "NE",
	"nevada":                       "NV",
	"new hampshire":                "NH",
	"new jersey":                   "NJ",
	"new mexico":                   "NM",
	"new york":                     "NY",
	"north carolina":               "NC",
	"north dakota":                 "ND",
	"northern mariana islands":     "MP",
	"ohio":                         "OH",
	"oklahoma":                     "OK",
	"oregon":                       "OR",
	"pennsylvania":                 "PA",
	"puerto rico":                  "PR",
	"rhode island":                 "RI",
	"south carolina":               "SC",
	"south dakota":                 "SD",
	"tennessee":                    "TN",
	"texas":                        "TX",
	"united states virgin islands": "VI",
	"utah":                         "UT",
	"vermont":                      "VT",
	"virgin islands":               "VI",
	"virginia":                     "VA",
	"washington":                   "WA",
	"west virginia":                "WV",
	"wisconsin":                    "WI",
	"wyoming":                      "WY",
}

func CountryCode(value string) string {
	return lookupCode(countryCodes, value)
}

func StateCode(value string) string {
	return lookupCode(stateCodes, value)
}

func CountryAliases(code string) []string {
	return aliasesFor(countryCodes, code)
}

// aliasesFor collects every spelling of a place that maps to the given code.
// Spellings that differ only in case are the same spelling once the column
// is compared case-insensitively, so only one of each is kept.
func aliasesFor(codes map[string]string, code string) []string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil
	}
	set := map[string]struct{}{strings.ToLower(code): {}}
	for alias, c := range codes {
		if c == code {
			set[strings.ToLower(alias)] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for alias := range set {
		out = append(out, alias)
	}
	sort.Strings(out)
	return out
}

// StateAliases returns every spelling that resolves to the given US state
// code — the full name and the two-letter code. The catalogue stores a state
// as a full name, so a filter written against the code has to be widened the
// same way a country filter is.
func StateAliases(code string) []string {
	return aliasesFor(stateCodes, code)
}

// Aliases returns every spelling that resolves to the given country or state
// code, whichever kind of place it is.
func Aliases(kind string, code string) []string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "state", "province", "region":
		return StateAliases(code)
	default:
		return CountryAliases(code)
	}
}

var continentCodes = map[string]string{
	"africa":          "Africa",
	"asia":            "Asia",
	"europe":          "Europe",
	"north america":   "North America",
	"central america": "Central America",
	"south america":   "South America",
	"oceania":         "Oceania",
	"antarctica":      "Antarctica",
}

func MatchContinent(query string) []string {
	trimmed := strings.ToLower(strings.TrimSpace(query))
	if trimmed == "" {
		return nil
	}
	if code, ok := continentCodes[trimmed]; ok {
		return []string{code}
	}

	words := strings.Fields(trimmed)
	for i := len(words) - 1; i >= 0 && i >= len(words)-2; i-- {
		if code, ok := continentCodes[words[i]]; ok {
			return []string{code}
		}
		if i+1 < len(words) {
			if code, ok := continentCodes[words[i]+" "+words[i+1]]; ok {
				return []string{code}
			}
		}
	}
	return nil
}

func MatchCountryNames(query string) []string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}

	if code, ok := countryCodes[strings.ToLower(trimmed)]; ok {
		return CountryAliases(code)
	}

	words := strings.Fields(strings.ToLower(trimmed))
	for i := len(words) - 1; i >= 0 && i >= len(words)-2; i-- {
		word := words[i]
		if code, ok := countryCodes[word]; ok {
			return CountryAliases(code)
		}
	}

	return nil
}

func lookupCode(codes map[string]string, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}
	lowered := strings.ToLower(trimmed)
	if code, ok := codes[lowered]; ok {
		return code
	}
	if trimmed, ok := strings.CutPrefix(lowered, "the "); ok {
		if code, ok := codes[trimmed]; ok {
			return code
		}
	}
	return value
}
