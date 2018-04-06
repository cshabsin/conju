package conju

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"

	"google.golang.org/appengine/mail"
)

type MailHeaderInfo struct {
	To      []string
	Cc      []string
	Bcc     []string
	Subject string
}

// Renders the named mail template and returns the filled text, filled
// html, and filled subject line, or an error.
func renderMail(templatePrefix string, data interface{}, functions template.FuncMap, needSubject bool) (string, string, string, error) {
	file := "templates/email/" + templatePrefix + ".html"
	tpl, err := template.New("").Funcs(functions).ParseFiles(file)
	if err != nil {
		return "", "", "", err
	}
	var text bytes.Buffer
	if err := tpl.ExecuteTemplate(&text, templatePrefix+"_text", data); err != nil {
		return "", "", "", err
	}
	var html bytes.Buffer
	if err := tpl.ExecuteTemplate(&html, templatePrefix+"_html", data); err != nil {
		return text.String(), "", "", err
	}
	if needSubject {
		var subject bytes.Buffer
		if err := tpl.ExecuteTemplate(&subject, templatePrefix+"_subject", data); err != nil {
			return text.String(), html.String(), "", err
		}
		return text.String(), html.String(), subject.String(), nil
	}
	return text.String(), html.String(), "", nil
}

func handleSendMail(wr WrappedRequest) {
	wr.Request.ParseForm()
	emailTemplates, ok := wr.Request.Form["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		emailTemplates, ok = wr.Request.PostForm["emailTemplate"]
	}
	if !ok || len(emailTemplates) == 0 {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("Use ?emailTemplate=<templateName> with %s.", wr.URL.Path),
			http.StatusBadRequest)
		return
	}
	emailTemplate := emailTemplates[0]
	// TODO: What data do we send this?
	realizedInvitation := makeRealizedInvitation(wr.Context, *wr.LoginInfo.InvitationKey,
		*wr.LoginInfo.Invitation)
	emailData := map[string]interface{}{
		"Event":      wr.Event,
		"Invitation": realizedInvitation,
		"Person":     wr.LoginInfo.Person,
		"LoginLink":  makeLoginUrl(wr.LoginInfo.Person),
	}
	text, html, subject, err := renderMail(emailTemplate, emailData, nil, true)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
		return
	}
	distributorNames := make([]string, 0)
	for k, _ := range AllDistributors {
		distributorNames = append(distributorNames, k)
	}
	data := wr.MakeTemplateData(map[string]interface{}{
		"TemplateName":    emailTemplate,
		"Subject":         subject,
		"Body":            text,
		"HTMLBody":        template.HTML(html),
		"AllDistributors": distributorNames,
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
	var sender EmailSender
	sender = func(ctx context.Context, emailData map[string]interface{}, headerData MailHeaderInfo) error {
		return sendMail(ctx, emailTemplate, emailData, nil, headerData)
	}
	if err := distributor(wr, sender); err != nil {
		// Email distributors output info as they go, so don't issue an HTTP error.
		fmt.Fprintf(wr.ResponseWriter, "Error from email distributor: %v", err)
	}
}

func sendMail(ctx context.Context, templatePrefix string, data interface{},
	functions template.FuncMap, headerData MailHeaderInfo) error {
	text, html, subject, err := renderMail(templatePrefix, data, functions,
		/* needSubject = */ headerData.Subject != "")
	if headerData.Subject != "" {
		subject = headerData.Subject
	}
	if err != nil {
		return err
	}
	msg := &mail.Message{
		Sender:   "**** sender address ****",
		To:       headerData.To,
		Bcc:      []string{"**** sender address ****"},
		Subject:  subject,
		Body:     text,
		HTMLBody: html,
	}
	if err := mail.Send(ctx, msg); err != nil {
		return err
		// log.Errorf(ctx, "Couldn't send email: %v", err)
	}
	return nil
}
