package engine

import "github.com/google/uuid"

var PlatformCatalogID = uuid.MustParse("00000000-0000-4000-8000-000000000031")

type Catalog struct {
	ID                  uuid.UUID
	Version             string
	Currency            string
	FloorCentsPerHour   int
	TargetCentsPerHour  int
	TargetMultiplierBPS int
	Coefficients        map[string]float64
}

func DefaultCatalog() Catalog {
	return Catalog{
		ID:                  PlatformCatalogID,
		Version:             "v1",
		Currency:            "EUR",
		FloorCentsPerHour:   1800,
		TargetCentsPerHour:  3500,
		TargetMultiplierBPS: 11000,
		Coefficients:        defaultCoefficients(),
	}
}

func (c Catalog) Coeff(key string, fallback float64) float64 {
	if c.Coefficients == nil {
		return fallback
	}
	if v, ok := c.Coefficients[key]; ok {
		return v
	}
	return fallback
}

func defaultCoefficients() map[string]float64 {
	return map[string]float64{
		"mult.complexity.low":    0.9,
		"mult.complexity.medium": 1.0,
		"mult.complexity.high":   1.15,
		"mult.complexity.custom": 1.25,
		"mult.urgency.normal":    1.0,
		"mult.urgency.fast":      1.15,
		"mult.urgency.urgent":    1.3,

		"base.website.landing":            30,
		"base.website.corporate":          90,
		"base.website.ecommerce":          160,
		"adder.website.extra_page":        1,
		"adder.website.multilingual":      12,
		"adder.website.cms":               8,
		"adder.website.content_not_ready": 8,
		"adder.website.seo":               4,
		"adder.website.uiux":              22,
		"adder.website.branding":          25,
		"adder.website.revisions.medium":  8,
		"adder.website.revisiouns.high":   16,
		"adder.website.integration":       7,
		"clamp.website.landing.min":       20,
		"clamp.website.landing.max":       40,
		"clamp.website.corporate.min":     60,
		"clamp.website.corporate.max":     120,
		"clamp.website.ecommerce.min":     120,
		"clamp.website.ecommerce.max":     200,

		"base.design.small":               15,
		"base.design.medium":              30,
		"base.design.large":               60,
		"adder.design.wireframes":         8,
		"adder.design.hi_fi":              12,
		"adder.design.design_system":      20,
		"adder.design.prototype":          10,
		"adder.design.handoff":            6,
		"adder.desin.extra_screen":        0.75,
		"adder.design.extra_screen_after": 8,
		"adder.design.no_brand":           6,
		"adder.design.mobile":             8,
		"adder.design.revisions.medium":   6,
		"adder.design.revisions.high":     12,

		"base.branding.logo":                 20,
		"base.branding.identity":             40,
		"base.branding.full":                 70,
		"adder.branding.palette":             4,
		"adder.branding.typography":          4,
		"adder.branding.guidelines":          10,
		"adder.branding.social_kit":          8,
		"adder.branding.business_cards":      3,
		"adder.branding.research":            8,
		"adder.branding.stakeholders.3_5":    4,
		"adder.branding.stakeholders.6_plus": 10,
		"adder.branding.revisions.medium":    6,
		"adder.branding.revisions.high":      12,
		"mult.branding.refresh":              0.85,

		"base.custom_web.mvp":                     120,
		"base.custom_web.growth":                  200,
		"base.custom_web.complex":                 320,
		"adder.custom_web.platform.admin":         24,
		"adder.custom_web.platform.marketing":     16,
		"adder.custom_web.auth.email":             12,
		"adder.custom_web.auth.sso":               20,
		"adder_custom_web.auth.both":              28,
		"adder.custom_web.rbac":                   16,
		"adder.custom_web.payments.one_time":      20,
		"adder.custom_web.payments.subscriptions": 36,
		"adder.custom_web.payments.marketplace":   48,
		"adder.custom_web.integration":            10,
		"adder.custom_web.realtime":               18,
		"adder.custom_web.files":                  10,
		"adder.custom_web.i18n":                   14,
		"adder.custom_web.design":                 40,
		"adder.custom_web.content_not_ready":      8,
		"adder.custom_web.revisions.medium":       12,
		"adder.custom_web.revisions.high":         24,
		"adder.custom_web.devops.basic":           8,
		"adder.custom_web.devops.cicd":            20,

		"base.custom_mobile.mvp":                140,
		"base.custom_mobile.growth":             220,
		"base.custom_mobile.complex":            340,
		"adder.custom_mobile.platform.both":     16,
		"adder.custom_mobile.backend.existing":  12,
		"adder.custom_mobile.backend.new":       48,
		"adder.custom_mobile.auth.email":        12,
		"adder.custom_mobile.auth.sso":          20,
		"adder.custom_mobile.auth.both":         28,
		"adder.custom_mobile.offline":           20,
		"adder.custom_mobile.push":              12,
		"adder.custom_mobile.iap":               24,
		"adder.custom_mobile.store":             16,
		"adder.custom_mobile.design":            40,
		"adder.custom_mobile.device.camera":     8,
		"adder.custom_mobile.device.maps":       8,
		"adder.custom_mobile.device.bluetooth":  12,
		"adder.custom_mobile.device.biometrics": 6,
		"adder.custom_mobile.content_not_ready": 6,
		"adder.custom_mobile.revisions.medium":  12,
		"adder.custom_mobile.revisions.high":    24,
		"base.marketing.retainer.basic":         15,
		"base.marketing.retainer.medium":        30,
		"base.marketing.retainer.advanced":      60,
		"adder.marketing.channel":               3,
		"adder.marketing.channel_cap":           18,
		"adder.marketing.creative":              0.3,
		"adder.marketing.creative_cap":          20,
		"adder.marketing.landing_support":       6,
		"adder.marketing.ad_spend.1_5k":         4,
		"adder.marketing.ad_spend.5k_plus":      10,
		"base.marketing.project":                24,
		"adder.marketing.setup.pixel":           4,
		"adder.marketing.setup.landing":         10,
		"adder.marketing.setup.email":           8,
		"adder.marketing.setup.creatives":       6,
		"adder.marketing.project.week":          6,
	}
}
