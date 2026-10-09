package engine

import (
	"fmt"
	"math"
	"strings"
)

type hoursBreakdown struct {
	hours            float64
	drivers          []string
	expensiveFactors []string
	riskFactors      []string
}

func Estimate(input Input, catalog Catalog) (Result, error) {
	if err := validateInput(input); err != nil {
		return Result{}, err
	}

	var (
		bd       hoursBreakdown
		err      error
		retainer bool
	)

	switch input.Category {
	case CategoryWebsite:
		bd, err = hoursWebsite(input, catalog)
	case CategoryDesign:
		bd, err = hoursDesign(input, catalog)
	case CategoryBranding:
		bd, err = hoursBranding(input, catalog)
	case CategoryCustomWeb:
		bd, err = hoursCustomWeb(input, catalog)
	case CategoryCustomMobile:
		bd, err = hoursCustomMobile(input, catalog)
	case CategoryMarketing:
		retainer = input.Mode == ModeRetainer
		bd, err = hoursMarketing(input, catalog)
	default:
		return Result{}, fmt.Errorf("unknown category %q", input.Category)
	}
	if err != nil {
		return Result{}, err
	}

	complexityMult := catalog.Coeff("mult.complexity."+input.Complexity, 1)
	urgencyMult := catalog.Coeff("mult.urgency."+input.Urgency, 1)
	if input.Complexity != ComplexityMedium {
		bd.drivers = append(bd.drivers, "Complexity: "+input.Complexity)
	}
	if input.Urgency != UrgencyNormal {
		bd.drivers = append(bd.drivers, "Urgency: "+input.Urgency)
	}

	bd.hours *= complexityMult * urgencyMult

	// Website clamps apply after multipliers (matches legacy zyntera behavior).
	if input.Category == CategoryWebsite && input.Website != nil {
		wt := input.Website.WebsiteType
		min := catalog.Coeff("clamp.website."+wt+".min", 0)
		max := catalog.Coeff("clamp.website."+wt+".max", 0)
		if min > 0 && bd.hours < min {
			bd.hours = min
		}
		if max > 0 && bd.hours > max {
			bd.hours = max
		}
	}

	bd.hours = math.Max(8, math.Round(bd.hours))

	riskLevel, riskExtra := scoreRisk(input, bd.hours)
	bd.riskFactors = uniqueStrings(append(bd.riskFactors, riskExtra...))

	hours := bd.hours
	minCents := int(math.Round(hours * float64(catalog.FloorCentsPerHour)))
	recCents := int(math.Round(hours * float64(catalog.TargetCentsPerHour) * float64(catalog.TargetMultiplierBPS) / 10000))

	result := Result{
		CatalogVersion:        catalog.Version,
		Currency:              catalog.Currency,
		InternalBaseRateCents: catalog.FloorCentsPerHour,
		TargetRateCents:       catalog.TargetCentsPerHour,
		MinimumPriceCents:     minCents,
		RecommendedPriceCents: recCents,
		EURPerHour:            float64(recCents) / 100 / hours,
		RiskLevel:             riskLevel,
		Drivers:               bd.drivers,
		ExpensiveFactors:      bd.expensiveFactors,
		RiskFactors:           bd.riskFactors,
	}

	if retainer {
		result.HoursPerMonth = &hours
	} else {
		result.EstimatedHours = &hours
		days := timelineDays(hours, input.Urgency)
		result.EstimatedTimelineDays = &days
	}

	return result, nil
}

func validateInput(input Input) error {
	switch input.Category {
	case CategoryWebsite, CategoryDesign, CategoryBranding, CategoryCustomWeb, CategoryCustomMobile, CategoryMarketing:
	default:
		return fmt.Errorf("invalid category")
	}
	if input.Mode != ModeProject && input.Mode != ModeRetainer {
		return fmt.Errorf("invalid mode")
	}
	if input.Category != CategoryMarketing && input.Mode != ModeProject {
		return fmt.Errorf("category %s only supports project mode", input.Category)
	}
	switch input.Complexity {
	case ComplexityLow, ComplexityMedium, ComplexityHigh, ComplexityCustom:
	default:
		return fmt.Errorf("invalid complexity")
	}
	switch input.Urgency {
	case UrgencyNormal, UrgencyFast, UrgencyUrgent:
	default:
		return fmt.Errorf("invalid urgency")
	}
	return nil
}

func timelineDays(hours float64, urgency string) int {
	throughput := 30.0
	switch urgency {
	case UrgencyFast:
		throughput = 36
	case UrgencyUrgent:
		throughput = 42
	}
	return int(math.Max(3, math.Round((hours/throughput)*7)))
}

func scoreRisk(input Input, hours float64) (string, []string) {
	score := 0
	factors := []string{}

	if input.Complexity == ComplexityHigh {
		score += 2
	}
	if input.Complexity == ComplexityCustom {
		score += 3
	}
	if input.Urgency == UrgencyFast {
		score++
	}
	if input.Urgency == UrgencyUrgent {
		score += 2
	}
	if hours >= 120 {
		score += 2
	}
	if hours >= 200 {
		score += 2
	}
	if input.Website != nil && !input.Website.ContentReady {
		score++
		factors = append(factors, "Content not ready")
	}
	if input.Website != nil && len(input.Website.Integrations) >= 3 {
		score += 2
		factors = append(factors, "Many integrations")
	}
	if input.Marketing != nil && input.Marketing.CampaignComplexity == "high" {
		score += 2
		factors = append(factors, "High campaign complexity")
	}
	if input.Marketing != nil && input.Marketing.ReportingLevel == "advanced" {
		score++
		factors = append(factors, "Advanced reporting")
	}

	if score <= 2 {
		return RiskLow, factors
	}
	if score <= 5 {
		return RiskModerate, factors
	}
	return RiskHigh, factors
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
