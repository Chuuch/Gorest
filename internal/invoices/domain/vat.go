package domain

import (
	"strings"
	"unicode"
)

const (
	RegimeUntaxed       = "untaxed"
	RegimeStandard      = "standard"
	RegimeReverseCharge = "reverse_charge"
	RegimeOutsideScope  = "outside_scope"

	DefaultVATRateBPS = 2000
)

var euCountries = map[string]struct{}{
	"AT": {}, "BE": {}, "BG": {}, "HR": {}, "CY": {}, "CZ": {}, "DK": {},
	"ET": {}, "FI": {}, "FR": {}, "DE": {}, "GR": {}, "HU": {}, "IE": {},
	"IT": {}, "LV": {}, "LT": {}, "LU": {}, "MT": {}, "NL": {}, "PL": {},
	"PT": {}, "RO": {}, "SK": {}, "SI": {}, "ES": {}, "SE": {},
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

func IsEUCountry(country string) bool {
	_, ok := euCountries[NormalizeCountry(country)]
	return ok
}

func ResolveVATRegime(
	sellerVATID, sellerCountry, buyerVATID, buyerCountry string,
	defaultRateBPS int,
) (regime string, rateBPS int) {
	sellerVATID = NormalizeVATID(sellerVATID)
	buyerVATID = NormalizeVATID(buyerVATID)
	sellerCountry = NormalizeCountry(sellerCountry)
	buyerCountry = NormalizeCountry(buyerCountry)

	if sellerVATID == "" {
		return RegimeUntaxed, 0
	}

	if buyerCountry != "" && !IsEUCountry(buyerCountry) {
		return RegimeOutsideScope, 0
	}

	if buyerVATID != "" &&
		IsEUCountry(sellerCountry) &&
		IsEUCountry(buyerCountry) &&
		sellerCountry != buyerCountry {
		return RegimeReverseCharge, 0
	}

	if defaultRateBPS <= 0 {
		defaultRateBPS = DefaultVATRateBPS
	}
	return RegimeStandard, defaultRateBPS
}

func VATCents(subtotalCents, rateBPS int) int {
	if subtotalCents <= 0 || rateBPS <= 0 {
		return 0
	}
	return (subtotalCents*rateBPS + 5000) / 10000
}

func ApplyTax(invoice *Invoice) {
	switch invoice.VATRegime {
	case RegimeStandard:
		invoice.VATCents = VATCents(invoice.SubtotalCents, invoice.VATRateBPS)
	default:
		invoice.VATRateBPS = 0
		invoice.VATCents = 0
	}
	invoice.TotalCents = invoice.SubtotalCents + invoice.VATCents
}

func (invoice *Invoice) BillingReady() error {
	if strings.TrimSpace(invoice.SellerLegalName) == "" ||
		strings.TrimSpace(invoice.SellerAddressLine1) == "" ||
		strings.TrimSpace(invoice.SellerCity) == "" ||
		strings.TrimSpace(invoice.SellerPostalCode) == "" ||
		NormalizeCountry(invoice.SellerCountry) == "" {
		return ErrBillingProfileIncomplete
	}

	if invoice.VATRegime != RegimeUntaxed {
		if NormalizeVATID(invoice.SellerVATID) == "" {
			return ErrBillingProfileIncomplete
		}
		if NormalizeCountry(invoice.BuyerCountry) == "" {
			return ErrBillingProfileIncomplete
		}
	}

	return nil
}

func VATNote(regime string) string {
	switch regime {
	case RegimeReverseCharge:
		return "VAT reverse charge - customer accounts for VAT."
	case RegimeOutsideScope:
		return "Outside scope of VAT."
	default:
		return ""
	}
}
