package conju

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	text_template "text/template"

	"gopkg.in/sendgrid/sendgrid-go.v2"

	"google.golang.org/appengine/log"
	"google.golang.org/appengine/mail"
)

// MailHeaderInfo contains the header info for outgoing email, passed into sendMail.
type MailHeaderInfo struct {
	To      []string
	Cc      []string
	Bcc     []string
	Subject string
}

// Renders the named mail template and returns the filled text, filled
// html, and filled subject line, or an error.
func renderMail(wr WrappedRequest, templatePrefix string, data interface{}, needSubject bool) (string, string, string, error) {
	functionMap := template.FuncMap{
		"HasHousingPreference":        RealInvHasHousingPreference,
		"PronounString":               GetPronouns,
		"CollectiveAddressFirstNames": CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
		"DerefPeople":                 DerefPeople,
	}
	tpl, err := template.New("").Funcs(functionMap).ParseGlob("templates/email/*.html")
	if err != nil {
		return "", "", "", err
	}

	tpl, err = tpl.ParseGlob("templates/" + wr.Event.ShortName + "/email/*.html")
	if err != nil {
		return "", "", "", err
	}

	// Hard-code that we want the roomingInfo template available for now.
	tpl, err = tpl.ParseFiles("templates/roomingInfo.html")
	if err != nil {
		return "", "", "", err
	}

	textFunctionMap := text_template.FuncMap{
		"HasHousingPreference":        RealInvHasHousingPreference,
		"PronounString":               GetPronouns,
		"CollectiveAddressFirstNames": CollectiveAddressFirstNames,
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

func handleSendMail(wr WrappedRequest) {
	wr.Request.ParseForm()
	emailTemplates, ok := wr.Request.Form["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		emailTemplates, ok = wr.Request.PostForm["emailTemplate"]
	}
	if !ok || len(emailTemplates) == 0 {
		handleListMail(wr)
		return
	}
	emailTemplate := emailTemplates[0]
	// TODO: What data do we send this?
	realizedInvitation := makeRealizedInvitation(wr.Context, wr.LoginInfo.InvitationKey,
		wr.LoginInfo.Invitation)
	roomingInfo := getRoomingInfoWithInvitation(wr, wr.LoginInfo.Invitation, wr.LoginInfo.InvitationKey)
	emailData := map[string]interface{}{
		"Event":       wr.Event,
		"Invitation":  realizedInvitation,
		"Person":      wr.LoginInfo.Person,
		"LoginLink":   makeLoginUrl(wr.LoginInfo.Person),
		"RoomingInfo": roomingInfo,
		"Env":         wr.GetEnvForTemplates(),
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
	tpl := template.Must(template.ParseFiles("templates/main.html", "templates/sendEmail.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "sendEmail.html", data); err != nil {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("Rendering HTML display: %v", err),
			http.StatusInternalServerError)
		return
	}
}

func handleDoSendMail(wr WrappedRequest) {
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
	var senderFunc EmailSender
	senderFunc = func(ctx context.Context, emailData map[string]interface{}, headerData MailHeaderInfo) error {
		if _, ok := emailData["LoginLink"]; !ok {
			emailData["LoginLink"] = makeLoginUrl(emailData["Person"].(*Person))
		}
		if _, ok := emailData["Env"]; !ok {
			emailData["Env"] = wr.GetEnvForTemplates()
		}
		return sendMail(wr, emailTemplate, emailData, headerData)
	}
	if err := distributor.Distribute(wr, senderFunc); err != nil {
		// Email distributors output info as they go, so don't issue an HTTP error.
		fmt.Fprintf(wr.ResponseWriter, "Error from email distributor: %v", err)
	}
}

func handleListMail(wr WrappedRequest) {
	templateNames, err := filepath.Glob("templates/email/*.html")
	if err != nil {
		log.Errorf(wr.Context, "Error globbing email templates: %v", err)
	}
	for i := range templateNames {
		templateNames[i] = strings.TrimPrefix(templateNames[i], "templates/email/")
		templateNames[i] = strings.TrimSuffix(templateNames[i], ".html")
	}
	eventTemplateNames, err := filepath.Glob("templates/" + wr.Event.ShortName + "/email/*.html")
	if err != nil {
		log.Errorf(wr.Context, "Error globbing event email templates: %v", err)
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
		log.Errorf(wr.Context, "%v", err)
	}
}

func makeSendMailLink(templateName string) string {
	return "/sendMail?emailTemplate=" + templateName
}

func sendMail(wr WrappedRequest, templatePrefix string, data interface{},
	headerData MailHeaderInfo) error {
	text, html, subject, err := renderMail(wr, templatePrefix, data,
		/* needSubject = */ headerData.Subject == "")
	if headerData.Subject != "" {
		subject = headerData.Subject
	}
	if err != nil {
		log.Errorf(wr.Context, "Error rendering mail: %v", err)
		return err
	}
	message := sendgrid.NewMail()
	for _, to := range headerData.To {
		message.AddTo(to)
	}
	message.AddBcc(wr.GetBccAddress())
	message.SetSubject(subject)
	message.SetHTML(html)
	message.SetText(text)
	message.SetFrom(wr.GetSenderAddress())
	wr.GetEmailClient().Send(message)
	return nil
}

func sendErrorMail(wr WrappedRequest, message string) {
	msg := mail.Message{
		Sender:  wr.GetSenderAddress(),
		To:      []string{wr.GetErrorAddress()},
		Subject: "[conju] Runtime error report",
		Body:    message,
	}
	if err := mail.Send(wr.Context, &msg); err != nil {
		log.Errorf(wr.Context, "Error sending error mail: %v", err)
	}
}
