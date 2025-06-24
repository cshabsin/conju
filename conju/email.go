package conju

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/cshabsin/conju/model/message"
	"github.com/cshabsin/conju/model/person"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

var emailFunctionMap = template.FuncMap{
	"HasHousingPreference":        RealInvHasHousingPreference,
	"PronounString":               person.GetPronouns,
	"CollectiveAddressFirstNames": person.CollectiveAddressFirstNames,
	"SharerName":                  MakeSharerName,
	"DerefPeople":                 DerefPeople,
}

// Renders the named mail template and returns the filled text, filled
// html, and filled subject line, or an error.
func renderMail(ctx context.Context, wr WrappedRequest, templatePrefix string, data any, needSubject bool) (string, string, string, error) {
	htmlTpl, textTpl, err := message.GetTemplates(ctx, wr.Event.Key, emailFunctionMap)
	if err != nil {
		return "", "", "", fmt.Errorf("error getting templates: %w", err)
	}

	// Hard-code that we want the roomingInfo template available for now.
	htmlTpl, err = htmlTpl.ParseFiles("templates/roomingInfo.html")
	if err != nil {
		return "", "", "", err
	}
	textTpl, err = textTpl.ParseFiles("templates/roomingInfo.html")
	if err != nil {
		return "", "", "", err
	}

	var text bytes.Buffer
	if err := textTpl.ExecuteTemplate(&text, templatePrefix+"_text", data); err != nil {
		return "", "", "", err
	}
	var htmlBuf bytes.Buffer
	if err := htmlTpl.ExecuteTemplate(&htmlBuf, templatePrefix+"_html", data); err != nil {
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

func handleEditMail(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()
	if wr.Request.Form["new"] != nil {
		// handleCreateMail(ctx, wr)
		fmt.Fprintf(wr.ResponseWriter, "Creating new email templates is not yet supported.")
		return
	}
	// emailTemplate=value or new=true
	emailTemplates, ok := wr.Request.Form["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		emailTemplates, ok = wr.Request.PostForm["emailTemplate"]
	}
	if !ok || len(emailTemplates) == 0 {
		handleListMail(ctx, wr)
		return
	}
	emailTemplate := emailTemplates[0]
	msg, err := message.FetchTemplateSource(ctx, wr.Event.Key, emailTemplate)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Error getting template %s: %v", emailTemplate, err),
			http.StatusInternalServerError)
		return
	}
	data := wr.MakeTemplateData(map[string]any{
		"EmailTemplate": emailTemplate,
		"TextBody":      msg.Plaintext,
		"HTMLBody":      msg.HTML,
		"Subject":       msg.Subject,
	})
	tplFiles := []string{"templates/main.html", "templates/editEmail.html"}
	webTpl := template.Must(template.ParseFiles(tplFiles...))
	if err := webTpl.ExecuteTemplate(wr.ResponseWriter, "editEmail.html", data); err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Error rendering edit mail page: %v", err),
			http.StatusInternalServerError)
		return
	}
}

func handleFetchMailTemplate(ctx context.Context, wr WrappedRequest) {

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
	emailData := map[string]any{
		"Event":       wr.Event,
		"Invitation":  realizedInvitation,
		"Person":      wr.LoginInfo.Person,
		"LoginLink":   makeLoginUrl(wr.LoginInfo.Person, true),
		"RoomingInfo": roomingInfo,
		"Env":         wr.GetEnvForTemplates(),
		"Unreserved":  unreserved,
	}
	text, html, subject, err := renderMail(ctx, wr, emailTemplate, emailData, true)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
		return
	}
	data := wr.MakeTemplateData(map[string]any{
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
	var senderFunc EmailSender = func(ctx context.Context, emailData map[string]any, headerData MailHeaderInfo) error {
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
		return sendMail(ctx, wr, emailTemplate, emailData, headerData)
	}
	if err := distributor.Distribute(ctx, wr, senderFunc); err != nil {
		// Email distributors output info as they go, so don't issue an HTTP error.
		fmt.Fprintf(wr.ResponseWriter, "Error from email distributor: %v", err)
	}
}

func handleListMail(ctx context.Context, wr WrappedRequest) {
	templates, err := message.ListTemplates(ctx, wr.Event.Key)
	if err != nil {
		log.Printf("Error listing templates: %v", err)
		fmt.Fprintf(wr.ResponseWriter, "Error listing templates: %v", err)
		return
	}

	functionMap := template.FuncMap{
		"makeSendMailLink": func(templateName string) string {
			return "/sendMail?emailTemplate=" + templateName
		},
		"makeEditMailLink": func(templateName string) string {
			return "/editMail?emailTemplate=" + templateName
		},
	}
	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/listEmail.html"))
	data := wr.MakeTemplateData(map[string]any{"Templates": templates})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "listEmail.html", data); err != nil {
		log.Println(err)
	}
}

const senders = "Dana Scott and Chris Shabsin"

func sendMail(ctx context.Context, wr WrappedRequest, templatePrefix string, data any,
	headerData MailHeaderInfo) error {
	text, html, subject, err := renderMail(ctx, wr, templatePrefix, data,
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
