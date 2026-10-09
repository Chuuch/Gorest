package engine

const (
	CategoryWebsite      = "website"
	CategoryDesign       = "design"
	CategoryBranding     = "branding"
	CategoryCustomWeb    = "custom_web"
	CategoryCustomMobile = "custom_mobile"
	CategoryMarketing    = "marketing"

	ModeProject  = "project"
	ModeRetainer = "retainer"

	ComplexityLow    = "low"
	ComplexityMedium = "medium"
	ComplexityHigh   = "high"
	ComplexityCustom = "custom"

	UrgencyNormal = "normal"
	UrgencyFast   = "fast"
	UrgencyUrgent = "urgent"

	RiskLow      = "low"
	RiskModerate = "moderate"
	RiskHigh     = "high"
)

type Input struct {
	Category   string `json:"category"`
	Mode       string `json:"mode"`
	Complexity string `json:"complexity"`
	Urgency    string `json:"urgency"`

	ProjectName    string `json:"project_name,omitempty"`
	RespondentName string `json:"respondent_name,omitempty"`
	CompanyName    string `json:"company_name,omitempty"`
	CustomerNotes  string `json:"customer_notes,omitempty"`
	InternalNotes  string `json:"internal_notes,omitempty"`

	Website      *WebsiteInput      `json:"website,omitempty"`
	Design       *DesignInput       `json:"design,omitempty"`
	Branding     *BrandingInput     `json:"branding,omitempty"`
	CustomWeb    *CustomWebInput    `json:"custom_web,omitempty"`
	CustomMobile *CustomMobileInput `json:"custom_mobile,omitempty"`
	Marketing    *MarketingInput    `json:"marketing,omitempty"`
}

type WebsiteInput struct {
	WebsiteType       string   `json:"website_type"`
	PageCount         int      `json:"page_count"`
	Multilingual      bool     `json:"multilingual"`
	CMSRequired       bool     `json:"cms_required"`
	ContentReady      bool     `json:"content_ready"`
	SEOSetup          bool     `json:"seo_setup"`
	UIUXIncluded      bool     `json:"uiux_included"`
	BrandingIncluded  bool     `json:"branding_included"`
	ExpectedRevisions string   `json:"expected_revisions"`
	Integrations      []string `json:"integrations"`
}

type DesignInput struct {
	Deliverables         []string `json:"deliverables"`
	ScreenCount          int      `json:"screen_count"`
	BrandGuidelinesExist bool     `json:"brand_guidelines_exist"`
	IncludesMobile       bool     `json:"includes_mobile"`
	IncludesDesktop      bool     `json:"includes_desktop"`
	DesignSize           string   `json:"design_size"`
	ExpectedRevisions    string   `json:"expected_revisions"`
}

type BrandingInput struct {
	Package            string   `json:"package"`
	Deliverables       []string `json:"deliverables"`
	HasExistingBrand   bool     `json:"has_existing_brand"`
	CompetitorResearch bool     `json:"competitor_research"`
	StakeholderCount   string   `json:"stakeholder_count"`
	ExpectedRevisions  string   `json:"expected_revisions"`
}

type CustomWebInput struct {
	Platforms         []string `json:"platforms"`
	Auth              string   `json:"auth"`
	RolesPermissions  bool     `json:"roles_permissions"`
	Payments          string   `json:"payments"`
	Integrations      []string `json:"integrations"`
	Realtime          bool     `json:"realtime"`
	FileUploads       bool     `json:"file_uploads"`
	Multilingual      bool     `json:"multilingual"`
	DesignIncluded    bool     `json:"design_included"`
	ContentReady      bool     `json:"content_ready"`
	AppScale          string   `json:"app_scale"`
	ExpectedRevisions string   `json:"expected_revisions"`
	HostingDevops     string   `json:"hosting_devops"`
}

type CustomMobileInput struct {
	Platforms         []string `json:"platforms"`
	Backend           string   `json:"backend"`
	Auth              string   `json:"auth"`
	Offline           bool     `json:"offline"`
	PushNotifications bool     `json:"push_notifications"`
	PaymentsInApp     bool     `json:"payments_in_app"`
	StoreRelease      bool     `json:"store_release"`
	DesignIncluded    bool     `json:"design_included"`
	DeviceFeatures    []string `json:"device_features"`
	ContentReady      bool     `json:"content_ready"`
	AppScale          string   `json:"app_scale"`
	ExpectedRevisions string   `json:"expected_revisions"`
}

type MarketingInput struct {
	Channels           []string `json:"channels"`
	CreativesPerMonth  int      `json:"creatives_per_month"`
	CreativesOneShot   int      `json:"creatives_one_shot"`
	ReportingLevel     string   `json:"reporting_level"`
	CampaignComplexity string   `json:"campaign_complexity"`
	LandingPageSupport bool     `json:"landing_page_support"`
	AdSpendBand        string   `json:"ad_spend_band"`
	CampaignGoal       string   `json:"campaign_goal"`
	DurationWeeks      int      `json:"duration_weeks"`
	SetupIncludes      []string `json:"setup_includes"`
}

type Result struct {
	CatalogVersion        string   `json:"catalog_version"`
	Currency              string   `json:"currency"`
	EstimatedHours        *float64 `json:"estimated_hours,omitempty"`
	HoursPerMonth         *float64 `json:"hours_per_month,omitempty"`
	EstimatedTimelineDays *int     `json:"estimated_timeline_days,omitempty"`
	InternalBaseRateCents int      `json:"internal_base_rate_cents"`
	TargetRateCents       int      `json:"target_rate_cents"`
	MinimumPriceCents     int      `json:"minimum_price_cents"`
	RecommendedPriceCents int      `json:"recommended_price_cents"`
	EURPerHour            float64  `json:"eur_per_hour"`
	RiskLevel             string   `json:"risk_level"`
	Drivers               []string `json:"drivers"`
	ExpensiveFactors      []string `json:"expensive_factors"`
	RiskFactors           []string `json:"risk_factors"`
}
