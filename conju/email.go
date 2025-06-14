package conju

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	text_template "text/template"

	"github.com/cshabsin/conju/model/person"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// MailHeaderInfo contains the header info for outgoing email, passed into sendMail.
type MailHeaderInfo struct {
	To      []string
	Cc      []string
	Bcc     []string
	Subject string

	BccSelf bool
}

// Renders the named mail template and returns the filled text, filled
// html, and filled subject line, or an error.
func renderMail(wr WrappedRequest, templatePrefix string, data interface{}, needSubject bool) (string, string, string, error) {
	functionMap := template.FuncMap{
		"HasHousingPreference":        RealInvHasHousingPreference,
		"PronounString":               person.GetPronouns,
		"CollectiveAddressFirstNames": person.CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
		"DerefPeople":                 DerefPeople,
	}
	tpl, err := template.New("").Funcs(functionMap).ParseGlob("templates/email/*.html")
	if err != nil {
		return "", "", "", err
	}

	tpl, err = tpl.ParseGlob("templates/" + wr.Event.ShortName + "/email/*.html")
	if err != nil {
		return "", "", "", fmt.Errorf("parsing templates %s: %v", "templates/"+wr.Event.ShortName+"/email/*.html", err)
	}

	// Hard-code that we want the roomingInfo template available for now.
	tpl, err = tpl.ParseFiles("templates/roomingInfo.html")
	if err != nil {
		return "", "", "", err
	}

	textFunctionMap := text_template.FuncMap{
		"HasHousingPreference":        RealInvHasHousingPreference,
		"PronounString":               person.GetPronouns,
		"CollectiveAddressFirstNames": person.CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
		"DerefPeople":                 DerefPeople,
	}
	textTpl, err := text_template.New("").Funcs(textFunctionMap).ParseGlob("templates/email/*.html")
	if err != nil {
		return "", "", "", err
	}
	textTpl, err = textTpl.ParseGlob("templates/" + wr.Event.ShortName + "/email/*.html")
	if err != nil {
		return "", "", "", err
	}

	// Hard-code that we want the roomingInfo template available for now.
	textTpl, err = textTpl.ParseFiles("templates/roomingInfo.html")
	if err != nil {
		return "", "", "", err
	}

	var text bytes.Buffer
	if err := textTpl.ExecuteTemplate(&text, templatePrefix+"_text", data); err != nil {
		return "", "", "", err
	}
	var htmlBuf bytes.Buffer
	if err := tpl.ExecuteTemplate(&htmlBuf, templatePrefix+"_html", data); err != nil {
		return text.String(), "", "", err
	}
	if needSubject {
		var subject bytes.Buffer
		if err := textTpl.ExecuteTemplate(&subject, templatePrefix+"_subject", data); err != nil {
			return text.String(), htmlBuf.String(), "", err
		}
		return text.String(), htmlBuf.String(), subject.String(), nil
	}
	return text.String(), htmlBuf.String(), "", nil
}

func handleSendMail(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()
	emailTemplates, ok := wr.Request.Form["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		emailTemplates, ok = wr.Request.PostForm["emailTemplate"]
	}
	if !ok || len(emailTemplates) == 0 {
		handleListMail(ctx, wr)
		return
	}
	handleMailPage(ctx, wr, emailTemplates[0], "sendEmail.html")
}

func handleViewMyInvitation(ctx context.Context, wr WrappedRequest) {
	handleMailPage(ctx, wr, "initial_invitation", "viewMyInvitation.html")
}

func handleMailPage(ctx context.Context, wr WrappedRequest, emailTemplate, htmlTemplate string) {
	// TODO: What data do we send this?
	realizedInvitation := makeRealizedInvitation(ctx, wr.LoginInfo.InvitationKey,
		wr.LoginInfo.Invitation)
	roomingInfo := getRoomingInfoWithInvitation(ctx, wr, wr.LoginInfo.Invitation, wr.LoginInfo.InvitationKey)
	var unreserved []BuildingRoom
	if roomingInfo != nil {
		for _, booking := range roomingInfo.InviteeBookings {
			if !booking.ReservationMade {
				unreserved = append(unreserved, BuildingRoom{booking.Room, booking.Building})
			}
		}
	}
	emailData := map[string]interface{}{
		"Event":       wr.Event,
		"Invitation":  realizedInvitation,
		"Person":      wr.LoginInfo.Person,
		"LoginLink":   makeLoginUrl(wr.LoginInfo.Person, true),
		"RoomingInfo": roomingInfo,
		"Env":         wr.GetEnvForTemplates(),
		"Unreserved":  unreserved,
	}
	text, html, subject, err := renderMail(wr, emailTemplate, emailData, true)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
		return
	}
	data := wr.MakeTemplateData(map[string]interface{}{
		"TemplateName":    emailTemplate,
		"Subject":         subject,
		"Body":            text,
		"HTMLBody":        template.HTML(html),
		"AllDistributors": AllDistributors,
	})
	tpl, err := template.ParseFiles("templates/main.html", "templates/"+htmlTemplate)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Parsing files: %v", err),
			http.StatusInternalServerError)
		return
	}
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, htmlTemplate, data); err != nil {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("Rendering HTML display: %v", err),
			http.StatusInternalServerError)
		return
	}
}

func handleDoSendMail(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()
	emailTemplates, ok := wr.Request.PostForm["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("%s issued without emailTemplate?", wr.URL.Path),
			http.StatusBadRequest)
		return
	}
	emailTemplate := emailTemplates[0]
	distributors, ok := wr.Request.PostForm["distributor"]
	if !ok || len(distributors) == 0 {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("%s issued without distributor?", wr.URL.Path),
			http.StatusBadRequest)
		return
	}
	distributorName := distributors[0]
	distributor, ok := AllDistributors[distributorName]
	if !ok {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("Bad distributor name: %s", distributorName),
			http.StatusBadRequest)
		return
	}
	bccSelf := wr.Request.PostForm.Get("bccSelf") == "1"
	var senderFunc EmailSender = func(ctx context.Context, emailData map[string]interface{}, headerData MailHeaderInfo) error {
		p := emailData["Person"].(*person.Person)
		if _, ok := emailData["LoginLink"]; !ok {
			emailData["LoginLink"] = makeLoginUrl(p, true)
		}
		if _, ok := emailData["Env"]; !ok {
			emailData["Env"] = wr.GetEnvForTemplates()
		}

		roomingAndCostInfo := emailData["RoomingInfo"].(*RoomingAndCostInfo)
		var unreserved []BuildingRoom
		if roomingAndCostInfo != nil {
			for _, booking := range roomingAndCostInfo.InviteeBookings {
				if !booking.ReservationMade {
					unreserved = append(unreserved, BuildingRoom{booking.Room, booking.Building})
				}
			}
		}
		emailData["Unreserved"] = unreserved
		headerData.BccSelf = bccSelf
		return sendMail(wr, emailTemplate, emailData, headerData)
	}
	if err := distributor.Distribute(ctx, wr, senderFunc); err != nil {
		// Email distributors output info as they go, so don't issue an HTTP error.
		fmt.Fprintf(wr.ResponseWriter, "Error from email distributor: %v", err)
	}
}

func handleListMail(ctx context.Context, wr WrappedRequest) {
	templateNames, err := filepath.Glob("templates/email/*.html")
	if err != nil {
		log.Printf("Error globbing email templates: %v", err)
	}
	for i := range templateNames {
		templateNames[i] = strings.TrimPrefix(templateNames[i], "templates/email/")
		templateNames[i] = strings.TrimSuffix(templateNames[i], ".html")
	}
	eventTemplateNames, err := filepath.Glob("templates/" + wr.Event.ShortName + "/email/*.html")
	if err != nil {
		log.Printf("Error globbing event email templates: %v", err)
	}
	for i := range eventTemplateNames {
		eventTemplateNames[i] = strings.TrimPrefix(eventTemplateNames[i], "templates/"+wr.Event.ShortName+"/email/")
		eventTemplateNames[i] = strings.TrimSuffix(eventTemplateNames[i], ".html")
	}

	templateNames = append(templateNames, eventTemplateNames...)
	functionMap := template.FuncMap{
		"makeSendMailLink": makeSendMailLink,
	}
	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/listEmail.html"))
	data := wr.MakeTemplateData(map[string]interface{}{"Templates": templateNames})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "listEmail.html", data); err != nil {
		log.Println(err)
	}
}

func makeSendMailLink(templateName string) string {
	return "/sendMail?emailTemplate=" + templateName
}

const senders = "Dana Scott and Chris Shabsin"

func sendMail(wr WrappedRequest, templatePrefix string, data interface{},
	headerData MailHeaderInfo) error {
	text, html, subject, err := renderMail(wr, templatePrefix, data,
		/* needSubject = */ headerData.Subject == "")
	if headerData.Subject != "" {
		subject = headerData.Subject
	}
	if err != nil {
		log.Printf("Error rendering mail: %v", err)
		return err
	}
	bccPers := mail.NewPersonalization()
	if headerData.BccSelf {
		bccPers.AddBCCs(mail.NewEmail("", wr.GetBccAddress()))
	}

	// TODO(cshabsin): get string name from somewhere environmental?
	message := &mail.SGMailV3{
		From:    mail.NewEmail(senders, wr.GetSenderAddress()),
		Subject: subject,
		Content: []*mail.Content{
			mail.NewContent("text/plain", text),
			mail.NewContent("text/html", html),
		},
		Personalizations: []*mail.Personalization{
			ToListPersonalization(wr, headerData.To),
		},
	}

	log.Printf("sending mail to %v: %v", headerData.To, message)
	if resp, err := wr.GetEmailClient().Send(message); err != nil {
		log.Printf("sendgrid.Send got err: %v, %v", resp, err)
	} else {
		log.Printf("sendgrid.Send got resp: %v", resp)
	}
	return nil
}

func ToListPersonalization(wr WrappedRequest, to []string) *mail.Personalization {
	mailPersonalizations := mail.NewPersonalization()
	for _, to := range to {
		mailPersonalizations.AddTos(mail.NewEmail("", to))
	}
	mailPersonalizations.AddBCCs(mail.NewEmail("", wr.GetBccAddress()))
	return mailPersonalizations
}

func ToPersonalization(name, addr string) *mail.Personalization {
	mailPersonalizations := mail.NewPersonalization()
	mailPersonalizations.AddTos(mail.NewEmail(name, addr))
	return mailPersonalizations
}

func sendErrorMail(wr WrappedRequest, message string) {
	mailPersonalizations := mail.NewPersonalization()
	mailPersonalizations.AddTos(mail.NewEmail("Errors", wr.GetErrorAddress()))
	msg := &mail.SGMailV3{
		From:    mail.NewEmail(senders, wr.GetSenderAddress()),
		Subject: "[conju] Runtime error report",
		Content: []*mail.Content{
			mail.NewContent("text/plain", message),
		},
		Personalizations: []*mail.Personalization{mailPersonalizations},
	}
	if _, err := wr.GetEmailClient().Send(msg); err != nil {
		log.Printf("Error sending error mail: %v", err)
	}
}
