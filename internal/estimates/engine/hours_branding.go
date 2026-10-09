package engine

import "fmt"

func hoursBranding(input Input, catalog Catalog) (hoursBreakdown, error) {
	if input.Branding == nil {
		return hoursBreakdown{}, fmt.Errorf("branding scope is required")
	}
	b := input.Branding
	bd := hoursBreakdown{}

	pkg := b.Package
	if pkg == "" {
		pkg = "identity"
	}
	hours := catalog.Coeff("base.branding."+pkg, 40)
	bd.drivers = append(bd.drivers, "Branding package: "+pkg)

	for _, del := range b.Deliverables {
		switch del {
		case "palette":
			hours += catalog.Coeff("adder.branding.palette", 4)
		case "typography":
			hours += catalog.Coeff("adder.branding.typography", 4)
		case "guidelines", "guidelines_pdf":
			hours += catalog.Coeff("adder.branding.guidelines", 10)
		case "social_kit":
			hours += catalog.Coeff("adder.branding.social_kit", 8)
		case "business_cards":
			hours += catalog.Coeff("adder.branding.business_cards", 3)
		}
	}
	if b.CompetitorResearch {
		hours += catalog.Coeff("adder.branding.research", 8)
		bd.expensiveFactors = append(bd.expensiveFactors, "Competitor research")
	}
	switch b.StakeholderCount {
	case "3-5":
		hours += catalog.Coeff("adder.branding.stakeholders.3_5", 4)
		bd.riskFactors = append(bd.riskFactors, "Multiple stakeholders")
	case "6+":
		hours += catalog.Coeff("adder.branding.stakeholders.6_plus", 10)
		bd.riskFactors = append(bd.riskFactors, "Many stakeholders")
	}
	if b.HasExistingBrand {
		hours *= catalog.Coeff("mult.branding.refresh", 0.85)
		bd.drivers = append(bd.drivers, "Brand refresh")
	}
	switch b.ExpectedRevisions {
	case "medium":
		hours += catalog.Coeff("adder.branding.revisions.medium", 6)
	case "high":
		hours += catalog.Coeff("adder.branding.revisions.high", 12)
	}

	bd.hours = hours
	return bd, nil
}
