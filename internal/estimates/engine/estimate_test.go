package engine_test

import (
	"testing"

	"github.com/chuuch/gorest/internal/estimates/engine"
	"github.com/stretchr/testify/require"
)

func TestEstimate_WebsiteCorporate(t *testing.T) {
	catalog := engine.DefaultCatalog()
	contentReady := true

	result, err := engine.Estimate(engine.Input{
		Category:   engine.CategoryWebsite,
		Mode:       engine.ModeProject,
		Complexity: engine.ComplexityMedium,
		Urgency:    engine.UrgencyNormal,
		Website: &engine.WebsiteInput{
			WebsiteType:       "corporate",
			PageCount:         6,
			CMSRequired:       true,
			ContentReady:      contentReady,
			SEOSetup:          true,
			ExpectedRevisions: "medium",
		},
	}, catalog)
	require.NoError(t, err)
	require.NotNil(t, result.EstimatedHours)
	require.Greater(t, *result.EstimatedHours, 60.0)
	require.LessOrEqual(t, *result.EstimatedHours, 120.0)
	require.Equal(t, "EUR", result.Currency)
	require.Greater(t, result.RecommendedPriceCents, result.MinimumPriceCents)
	require.NotNil(t, result.EstimatedTimelineDays)
}

func TestEstimate_WebsiteLandingClamp(t *testing.T) {
	catalog := engine.DefaultCatalog()

	result, err := engine.Estimate(engine.Input{
		Category:   engine.CategoryWebsite,
		Mode:       engine.ModeProject,
		Complexity: engine.ComplexityCustom,
		Urgency:    engine.UrgencyUrgent,
		Website: &engine.WebsiteInput{
			WebsiteType:       "landing",
			PageCount:         20,
			Multilingual:      true,
			CMSRequired:       true,
			ContentReady:      false,
			SEOSetup:          true,
			UIUXIncluded:      true,
			BrandingIncluded:  true,
			ExpectedRevisions: "high",
			Integrations:      []string{"crm", "payments", "booking", "chat"},
		},
	}, catalog)
	require.NoError(t, err)
	require.NotNil(t, result.EstimatedHours)
	require.Equal(t, 40.0, *result.EstimatedHours) // clamp max
}

func TestEstimate_MarketingRetainer(t *testing.T) {
	catalog := engine.DefaultCatalog()

	result, err := engine.Estimate(engine.Input{
		Category:   engine.CategoryMarketing,
		Mode:       engine.ModeRetainer,
		Complexity: engine.ComplexityMedium,
		Urgency:    engine.UrgencyNormal,
		Marketing: &engine.MarketingInput{
			Channels:           []string{"facebook_ads", "google_ads"},
			CreativesPerMonth:  8,
			ReportingLevel:     "standard",
			CampaignComplexity: "medium",
			LandingPageSupport: true,
		},
	}, catalog)
	require.NoError(t, err)
	require.NotNil(t, result.HoursPerMonth)
	require.Nil(t, result.EstimatedHours)
	require.Nil(t, result.EstimatedTimelineDays)
	require.Greater(t, *result.HoursPerMonth, 20.0)
}

func TestEstimate_CustomMobileAssumesReactNative(t *testing.T) {
	catalog := engine.DefaultCatalog()

	result, err := engine.Estimate(engine.Input{
		Category:   engine.CategoryCustomMobile,
		Mode:       engine.ModeProject,
		Complexity: engine.ComplexityMedium,
		Urgency:    engine.UrgencyNormal,
		CustomMobile: &engine.CustomMobileInput{
			Platforms:         []string{"ios", "android"},
			Backend:           "new",
			Auth:              "email",
			ContentReady:      true,
			AppScale:          "mvp",
			ExpectedRevisions: "medium",
		},
	}, catalog)
	require.NoError(t, err)
	require.NotNil(t, result.EstimatedHours)
	require.Contains(t, result.Drivers, "Tech: react_native")
}

func TestEstimate_RejectsInvalidCategory(t *testing.T) {
	_, err := engine.Estimate(engine.Input{
		Category:   "nope",
		Mode:       engine.ModeProject,
		Complexity: engine.ComplexityMedium,
		Urgency:    engine.UrgencyNormal,
	}, engine.DefaultCatalog())
	require.Error(t, err)
}
