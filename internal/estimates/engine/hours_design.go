package engine

import "fmt"

func hoursDesign(input Input, catalog Catalog) (hoursBreakdown, error) {
	if input.Design == nil {
		return hoursBreakdown{}, fmt.Errorf("design scope is required")
	}
	d := input.Design
	bd := hoursBreakdown{}

	size := d.DesignSize
	if size == "" {
		size = "medium"
	}
	hours := catalog.Coeff("base.design."+size, 30)
	bd.drivers = append(bd.drivers, "Design size: "+size)

	for _, del := range d.Deliverables {
		switch del {
		case "wireframes":
			hours += catalog.Coeff("adder.design.wireframes", 8)
		case "hi_fi", "hi-fi":
			hours += catalog.Coeff("adder.design.hi_fi", 12)
		case "design_system":
			hours += catalog.Coeff("adder.design.design_system", 20)
		case "prototype":
			hours += catalog.Coeff("adder.design.prototype", 10)
		case "handoff":
			hours += catalog.Coeff("adder.design.handoff", 6)
		}
	}

	after := int(catalog.Coeff("adder.design.extra_screen_after", 8))
	if d.ScreenCount > after {
		extra := d.ScreenCount - after
		hours += float64(extra) * catalog.Coeff("adder.design.extra_screen", 0.75)
		bd.expensiveFactors = append(bd.expensiveFactors, fmt.Sprintf("Extra screens (+%d)", extra))
	}
	if !d.BrandGuidelinesExist {
		hours += catalog.Coeff("adder.design.no_brand", 6)
		bd.riskFactors = append(bd.riskFactors, "No brand guidelines")
	}
	if d.IncludesMobile {
		hours += catalog.Coeff("adder.design.mobile", 8)
		bd.expensiveFactors = append(bd.expensiveFactors, "Mobile design")
	}
	switch d.ExpectedRevisions {
	case "medium":
		hours += catalog.Coeff("adder.design.revisions.medium", 6)
	case "high":
		hours += catalog.Coeff("adder.design.revisions.high", 12)
		bd.expensiveFactors = append(bd.expensiveFactors, "High revisions")
	}

	bd.hours = hours
	return bd, nil
}
