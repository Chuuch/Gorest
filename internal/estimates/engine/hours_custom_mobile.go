package engine

import "fmt"

func hoursCustomMobile(input Input, catalog Catalog) (hoursBreakdown, error) {
	if input.CustomMobile == nil {
		return hoursBreakdown{}, fmt.Errorf("custom_mobile scope is required")
	}
	m := input.CustomMobile
	bd := hoursBreakdown{}

	scale := m.AppScale
	if scale == "" {
		scale = "mvp"
	}
	hours := catalog.Coeff("base.custom_mobile."+scale, 140)
	bd.drivers = append(bd.drivers, "App scale: "+scale)
	bd.drivers = append(bd.drivers, "Tech: react_native")

	hasIOS, hasAndroid := false, false
	for _, p := range m.Platforms {
		switch p {
		case "ios":
			hasIOS = true
		case "android":
			hasAndroid = true
		case "both":
			hasIOS, hasAndroid = true, true
		}
	}
	if hasIOS && hasAndroid {
		hours += catalog.Coeff("adder.custom_mobile.platform.both", 16)
		bd.expensiveFactors = append(bd.expensiveFactors, "iOS + Android")
	}

	switch m.Backend {
	case "existing":
		hours += catalog.Coeff("adder.custom_mobile.backend.existing", 12)
		bd.expensiveFactors = append(bd.expensiveFactors, "Existing API")
	case "new":
		hours += catalog.Coeff("adder.custom_mobile.backend.new", 48)
		bd.expensiveFactors = append(bd.expensiveFactors, "New API")
	}

	switch m.Auth {
	case "email":
		hours += catalog.Coeff("adder.custom_mobile.auth.email", 12)
	case "sso":
		hours += catalog.Coeff("adder.custom_mobile.auth.sso", 20)
	case "both":
		hours += catalog.Coeff("adder.custom_mobile.auth.both", 28)
	}

	if m.Offline {
		hours += catalog.Coeff("adder.custom_mobile.offline", 20)
		bd.expensiveFactors = append(bd.expensiveFactors, "Offline support")
	}
	if m.PushNotifications {
		hours += catalog.Coeff("adder.custom_mobile.push", 12)
	}
	if m.PaymentsInApp {
		hours += catalog.Coeff("adder.custom_mobile.iap", 24)
		bd.expensiveFactors = append(bd.expensiveFactors, "In-app payments")
	}
	if m.StoreRelease {
		hours += catalog.Coeff("adder.custom_mobile.store", 16)
		bd.expensiveFactors = append(bd.expensiveFactors, "Store release")
	}
	if m.DesignIncluded {
		hours += catalog.Coeff("adder.custom_mobile.design", 40)
		bd.expensiveFactors = append(bd.expensiveFactors, "Design included")
	}
	if !m.ContentReady {
		hours += catalog.Coeff("adder.custom_mobile.content_not_ready", 6)
		bd.riskFactors = append(bd.riskFactors, "Content not ready")
	}

	for _, f := range m.DeviceFeatures {
		switch f {
		case "camera":
			hours += catalog.Coeff("adder.custom_mobile.device.camera", 8)
		case "maps":
			hours += catalog.Coeff("adder.custom_mobile.device.maps", 8)
		case "bluetooth":
			hours += catalog.Coeff("adder.custom_mobile.device.bluetooth", 12)
		case "biometrics":
			hours += catalog.Coeff("adder.custom_mobile.device.biometrics", 6)
		}
	}

	switch m.ExpectedRevisions {
	case "medium":
		hours += catalog.Coeff("adder.custom_mobile.revisions.medium", 12)
	case "high":
		hours += catalog.Coeff("adder.custom_mobile.revisions.high", 24)
		bd.expensiveFactors = append(bd.expensiveFactors, "High revisions")
	}

	bd.hours = hours
	return bd, nil
}
