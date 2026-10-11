package estimates

import (
	"fmt"

	"github.com/chuuch/gorest/internal/platform/mailer"
)

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
