package engine

import "fmt"

func hoursCustomWeb(input Input, catalog Catalog) (hoursBreakdown, error) {
	if input.CustomWeb == nil {
		return hoursBreakdown{}, fmt.Errorf("custom_web scope is required")
	}
	w := input.CustomWeb
	bd := hoursBreakdown{}

	scale := w.AppScale
	if scale == "" {
		scale = "mvp"
	}
	hours := catalog.Coeff("base.custom_web."+scale, 120)
	bd.drivers = append(bd.drivers, "App scale: "+scale)

	for _, p := range w.Platforms {
		switch p {
		case "admin":
			hours += catalog.Coeff("adder.custom_web.platform.admin", 24)
			bd.expensiveFactors = append(bd.expensiveFactors, "Admin panel")
		case "marketing":
			hours += catalog.Coeff("adder.custom_web.platform.marketing", 16)
			bd.expensiveFactors = append(bd.expensiveFactors, "Marketing site")
		}
	}
	switch w.Auth {
	case "email":
		hours += catalog.Coeff("adder.custom_web.auth.email", 12)
	case "sso":
		hours += catalog.Coeff("adder.custom_web.auth.sso", 20)
	case "both":
		hours += catalog.Coeff("adder.custom_web.auth.both", 28)
	}
	if w.RolesPermissions {
		hours += catalog.Coeff("adder.custom_web.rbac", 16)
		bd.expensiveFactors = append(bd.expensiveFactors, "Roles & permissions")
	}
	switch w.Payments {
	case "one_time":
		hours += catalog.Coeff("adder.custom_web.payments.one_time", 20)
	case "subscriptions":
		hours += catalog.Coeff("adder.custom_web.payments.subscriptions", 36)
	case "marketplace":
		hours += catalog.Coeff("adder.custom_web.payments.marketplace", 48)
	}
	if n := len(w.Integrations); n > 0 {
		hours += float64(n) * catalog.Coeff("adder.custom_web.integration", 10)
		bd.expensiveFactors = append(bd.expensiveFactors, fmt.Sprintf("Integrations × %d", n))
	}
	if w.Realtime {
		hours += catalog.Coeff("adder.custom_web.realtime", 18)
	}
	if w.FileUploads {
		hours += catalog.Coeff("adder.custom_web.files", 10)
	}
	if w.Multilingual {
		hours += catalog.Coeff("adder.custom_web.i18n", 14)
	}
	if w.DesignIncluded {
		hours += catalog.Coeff("adder.custom_web.design", 40)
		bd.expensiveFactors = append(bd.expensiveFactors, "Design included")
	}
	if !w.ContentReady {
		hours += catalog.Coeff("adder.custom_web.content_not_ready", 8)
		bd.riskFactors = append(bd.riskFactors, "Content not ready")
	}
	switch w.ExpectedRevisions {
	case "medium":
		hours += catalog.Coeff("adder.custom_web.revisions.medium", 12)
	case "high":
		hours += catalog.Coeff("adder.custom_web.revisions.high", 24)
	}
	switch w.HostingDevops {
	case "basic":
		hours += catalog.Coeff("adder.custom_web.devops.basic", 8)
	case "cicd":
		hours += catalog.Coeff("adder.custom_web.devops.cicd", 20)
	}

	bd.hours = hours
	return bd, nil
}
