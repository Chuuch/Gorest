package taxid

import (
	"strings"
	"unicode"
)

var EU = map[string]struct{}{
	"AT": {},
	"BE": {},
	"BG": {},
	"HR": {},
	"CY": {},
	"CZ": {},
	"DK": {},
	"EE": {},
	"FI": {},
	"FR": {},
	"DE": {},
	"GR": {},
	"HU": {},
	"IE": {},
	"IT": {},
	"LV": {},
	"LT": {},
	"LU": {},
	"MT": {},
	"NL": {},
	"PL": {},
	"PT": {},
	"RO": {},
	"SK": {},
	"SI": {},
	"ES": {},
	"SE": {},
}

func NormalizeCountry(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func NormalizeVATID(value string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func IsEU(country string) bool {
	_, ok := EU[NormalizeCountry(country)]
	return ok
}
