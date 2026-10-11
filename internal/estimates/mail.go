package estimates

import (
	"fmt"

	"github.com/chuuch/gorest/internal/platform/mailer"
)

func InviteMail(respondentName, categoryLabel, intakeURL string) (subject, text, html string, err error) {
	if respondentName == "" {
		respondentName = "there"
	}
	if categoryLabel == "" {
		categoryLabel = "your project"
	}

	subject = "Quick questionnaire about " + categoryLabel
	text = fmt.Sprintf(
		"Hi %s, \n\nThanks for your enquiry about %s.\n"+
			"Please fill in this short questionnaire so we can prepare an estimate:\n\n%s\n\n"+
			"This link expires in 14 days.\n",
		respondentName, categoryLabel, intakeURL,
	)

	html, err = mailer.RenderTransactional(mailer.Transactional{
		Heading:     "Tell us a bit more",
		Body:        fmt.Sprintf("Hi %s, thanks for your enquiry about %s. A short questionnaire will help us prepare an estimate.", respondentName, categoryLabel),
		ActionURL:   intakeURL,
		ActionLabel: "Open questionnaire",
		Expiry:      "This link expires in 14 days.",
	})
	if err != nil {
		return "", "", "", err
	}
	return subject, text, html, nil
}

func ConfirmationMail(respondentName, projectName string) (subject, text, html string, err error) {
	if respondentName == "" {
		respondentName = "there"
	}
	if projectName == "" {
		projectName = "your project"
	}

	subject = "We received your enquiry - Flourish"
	text = fmt.Sprintf("Hi %s, \n\nThanks - we received your enquiry for %s.\n"+
		"Our team will review it and follow up at the earliest convenience.\n", respondentName, projectName)

	html, err = mailer.RenderTransactional(mailer.Transactional{
		Heading: "Thanks - we got it!",
		Body:    fmt.Sprintf("Hi %s, we received your enquiry for %s. Our team will follow up at the earliest convenience", respondentName, projectName),
	})
	if err != nil {
		return "", "", "", err
	}
	return subject, text, html, nil
}

func CategoryLabel(category string) string {
	switch category {
	case "website":
		return "a website"
	case "design":
		return "design"
	case "branding":
		return "branding"
	case "custom_web":
		return "a custom web app"
	case "custom_mobile":
		return "a mobile app"
	case "marketing":
		return "marketing"
	default:
		return "your project"
	}
}
