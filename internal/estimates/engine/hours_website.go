package engine

import "fmt"

func hoursWebsite(input Input, catalog Catalog) (hoursBreakdown, error) {
	if input.Website == nil {
		return hoursBreakdown{}, fmt.Errorf("website scope is required")
	}
	w := input.Website
	bd := hoursBreakdown{}

	baseKey := "base.website.corporate"
	switch w.WebsiteType {
	case "landing":
		baseKey = "base.website.landing"
	case "corporate":
		baseKey = "base.website.corporate"
	case "ecommerce":
		baseKey = "base.website.ecommerce"
	default:
		return hoursBreakdown{}, fmt.Errorf("invalid website_type")
	}

	hours := catalog.Coeff(baseKey, 90)
	bd.drivers = append(bd.drivers, "Website type: "+w.WebsiteType)

	if extra := w.PageCount - 5; extra > 0 {
		hours += float64(extra) * catalog.Coeff("adder.website.extra_page", 1)
		bd.expensiveFactors = append(bd.expensiveFactors, fmt.Sprintf("Extra pages (+%d)", extra))
	}
	if w.Multilingual {
		hours += catalog.Coeff("adder.website.multilingual", 12)
		bd.expensiveFactors = append(bd.expensiveFactors, "Multilingual")
		bd.riskFactors = append(bd.riskFactors, "Multilingual content/QA")
	}
	if w.CMSRequired {
		hours += catalog.Coeff("adder.website.cms", 8)
		bd.expensiveFactors = append(bd.expensiveFactors, "CMS required")
	}
	if !w.ContentReady {
		hours += catalog.Coeff("adder.website.content_not_ready", 8)
		bd.riskFactors = append(bd.riskFactors, "Content not ready")
	}
	if w.SEOSetup {
		hours += catalog.Coeff("adder.website.seo", 4)
		bd.expensiveFactors = append(bd.expensiveFactors, "SEO setup")
	}
	if w.UIUXIncluded {
		hours += catalog.Coeff("adder.website.uiux", 22)
		bd.expensiveFactors = append(bd.expensiveFactors, "UI/UX included")
	}
	if w.BrandingIncluded {
		hours += catalog.Coeff("adder.website.branding", 25)
		bd.expensiveFactors = append(bd.expensiveFactors, "Branding included")
	}
	switch w.ExpectedRevisions {
	case "medium":
		hours += catalog.Coeff("adder.website.revisions.medium", 8)
		bd.riskFactors = append(bd.riskFactors, "Medium revisions")
	case "high":
		hours += catalog.Coeff("adder.website.revisions.high", 16)
		bd.expensiveFactors = append(bd.expensiveFactors, "High revisions")
		bd.riskFactors = append(bd.riskFactors, "High revisions")
	}
	if n := len(w.Integrations); n > 0 {
		hours += float64(n) * catalog.Coeff("adder.website.integration", 7)
		bd.expensiveFactors = append(bd.expensiveFactors, fmt.Sprintf("Integrations × %d", n))
	}

	bd.hours = hours
	return bd, nil
}
