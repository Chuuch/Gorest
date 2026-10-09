package engine

import (
	"fmt"
	"math"
)

func hoursMarketing(input Input, catalog Catalog) (hoursBreakdown, error) {
	if input.Marketing == nil {
		return hoursBreakdown{}, fmt.Errorf("marketing scope is required")
	}
	m := input.Marketing
	bd := hoursBreakdown{}

	if input.Mode == ModeRetainer {
		return hoursMarketingRetainer(m, catalog, bd)
	}
	return hoursMarketingProject(m, catalog, bd)
}

func hoursMarketingRetainer(m *MarketingInput, catalog Catalog, bd hoursBreakdown) (hoursBreakdown, error) {
	tier := marketingTier(m.ReportingLevel, m.CampaignComplexity)
	hours := catalog.Coeff("base.marketing.retainer."+tier, 30)
	bd.drivers = append(bd.drivers, "Marketing tier: "+tier)

	channelCount := len(m.Channels)
	if channelCount > 0 {
		add := math.Min(
			catalog.Coeff("adder.marketing.channel_cap", 18),
			float64(channelCount)*catalog.Coeff("adder.marketing.channel", 3),
		)
		hours += add
		bd.drivers = append(bd.drivers, fmt.Sprintf("Channels × %d", channelCount))
	}

	if m.CreativesPerMonth > 0 {
		add := math.Min(
			catalog.Coeff("adder.marketing.creative_cap", 20),
			float64(m.CreativesPerMonth)*catalog.Coeff("adder.marketing.creative", 0.3),
		)
		hours += add
		bd.expensiveFactors = append(bd.expensiveFactors, "Creatives volume")
	}

	if m.LandingPageSupport {
		hours += catalog.Coeff("adder.marketing.landing_support", 6)
		bd.expensiveFactors = append(bd.expensiveFactors, "Landing page support")
	}

	switch m.AdSpendBand {
	case "1-5k":
		hours += catalog.Coeff("adder.marketing.ad_spend.1_5k", 4)
	case "5k+":
		hours += catalog.Coeff("adder.marketing.ad_spend.5k_plus", 10)
		bd.expensiveFactors = append(bd.expensiveFactors, "High ad spend")
	}

	bd.hours = hours
	return bd, nil
}

func hoursMarketingProject(m *MarketingInput, catalog Catalog, bd hoursBreakdown) (hoursBreakdown, error) {
	hours := catalog.Coeff("base.marketing.project", 24)
	if m.CampaignGoal != "" {
		bd.drivers = append(bd.drivers, "Campaign goal: "+m.CampaignGoal)
	}

	channelCount := len(m.Channels)
	if channelCount > 0 {
		add := math.Min(
			catalog.Coeff("adder.marketing.channel_cap", 18),
			float64(channelCount)*catalog.Coeff("adder.marketing.channel", 3),
		)
		hours += add
		bd.drivers = append(bd.drivers, fmt.Sprintf("Channels × %d", channelCount))
	}

	if m.DurationWeeks > 0 {
		hours += float64(m.DurationWeeks) * catalog.Coeff("adder.marketing.project.week", 6)
		bd.drivers = append(bd.drivers, fmt.Sprintf("Duration %dw", m.DurationWeeks))
	}

	if m.CreativesOneShot > 0 {
		add := math.Min(
			catalog.Coeff("adder.marketing.creative_cap", 20),
			float64(m.CreativesOneShot)*catalog.Coeff("adder.marketing.creative", 0.3),
		)
		hours += add
		bd.expensiveFactors = append(bd.expensiveFactors, "Creatives pack")
	}

	for _, s := range m.SetupIncludes {
		switch s {
		case "pixel", "analytics":
			hours += catalog.Coeff("adder.marketing.setup.pixel", 4)
		case "landing", "landing_page":
			hours += catalog.Coeff("adder.marketing.setup.landing", 10)
		case "email", "email_sequences":
			hours += catalog.Coeff("adder.marketing.setup.email", 8)
		case "creatives", "creative_pack":
			hours += catalog.Coeff("adder.marketing.setup.creatives", 6)
		}
	}

	if m.LandingPageSupport {
		hours += catalog.Coeff("adder.marketing.landing_support", 6)
	}

	bd.hours = hours
	return bd, nil
}

func marketingTier(reportingLevel, campaignComplexity string) string {
	if reportingLevel == "advanced" || campaignComplexity == "high" {
		return "advanced"
	}
	if reportingLevel == "standard" || campaignComplexity == "medium" {
		return "medium"
	}
	return "basic"
}
