package conju

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/model/event"
	"github.com/cshabsin/conju/model/message"
	"github.com/cshabsin/conju/model/person"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/text/encoding/charmap"
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
func RenderMail(ctx context.Context, eventKey *datastore.Key, templatePrefix string, data any, needSubject bool) (string, string, string, error) {
	htmlTpl, textTpl, err := message.GetTemplates(ctx, eventKey, emailFunctionMap)
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

func handleSendMail(ctx context.Context, wr *WrappedRequest) {
	wr.Request.ParseForm()
	emailTemplates, ok := wr.Request.Form["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		emailTemplates, ok = wr.Request.PostForm["emailTemplate"]
	}
	if !ok || len(emailTemplates) == 0 {
		handleListMail(ctx, wr)
		return
	}
	key, err := datastore.DecodeKey(emailTemplates[0])
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Error decoding email template key %s: %v", emailTemplates[0], err),
			http.StatusBadRequest)
		return
	}
	msg, err := message.Get(ctx, key)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Error getting email template %s: %v", emailTemplates[0], err),
			http.StatusInternalServerError)
		return
	}
	handleMailPage(ctx, wr, msg.ShortName, "sendEmail.html")
}

func handleEditMail(ctx context.Context, wr *WrappedRequest) {
	wr.Request.ParseForm()
	isNew := false
	templateKey := ""
	shortName := ""
	textBody := ""
	htmlBody := ""
	subject := ""
	isGlobal := false
	if wr.Request.Form["new"] != nil {
		isNew = true
	} else {
		emailTemplates, ok := wr.Request.Form["emailTemplate"]
		if !ok || len(emailTemplates) == 0 {
			emailTemplates, ok = wr.Request.PostForm["emailTemplate"]
		}
		if !ok || len(emailTemplates) == 0 {
			handleListMail(ctx, wr)
			return
		}
		templateKey = emailTemplates[0]
		key, err := datastore.DecodeKey(templateKey)
		if err != nil {
			http.Error(wr.ResponseWriter, fmt.Sprintf("Error decoding email template key %s: %v", templateKey, err),
				http.StatusBadRequest)
			return
		}
		msg, err := message.Get(ctx, key)
		if err != nil {
			http.Error(wr.ResponseWriter, fmt.Sprintf("Error getting template %s: %v", templateKey, err),
				http.StatusInternalServerError)
			return
		}
		textBody = msg.Plaintext // TODO: convert to ISO-8859-1?
		htmlBody = msg.HTML
		subject = msg.Subject
		isGlobal = msg.Event == nil
	}
	data := wr.MakeTemplateData(map[string]any{
		"IsNew":       isNew,
		"IsGlobal":    isGlobal,
		"TemplateKey": templateKey,
		"ShortName":   shortName,
		"TextBody":    textBody,
		"HTMLBody":    htmlBody,
		"Subject":     subject,
	})
	tplFiles := []string{"templates/main.html", "templates/editEmail.html"}
	webTpl := template.Must(template.ParseFiles(tplFiles...))
	wr.ResponseWriter.Header().Set("Content-Type", "text/html")
	if err := webTpl.ExecuteTemplate(wr.ResponseWriter, "editEmail.html", data); err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Error rendering edit mail page: %v", err),
			http.StatusInternalServerError)
		return
	}
}

func handleSaveMail(ctx context.Context, wr *WrappedRequest) {
	wr.Request.ParseForm()
	emailTemplates, ok := wr.Request.PostForm["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("%s issued without emailTemplate?", wr.URL.Path),
			http.StatusBadRequest)
		return
	}
	emailTemplate := emailTemplates[0]
	textBody := wr.Request.PostForm.Get("textBody")
	htmlBody := wr.Request.PostForm.Get("htmlBody")
	subject := wr.Request.PostForm.Get("subject")

	eventKey := wr.Event.Key
	var keyForSanityCheck *datastore.Key
	if wr.Request.PostForm.Get("isGlobal") == "1" {
		eventKey = nil // Global templates have no event key.
	}
	if wr.Request.PostForm.Get("isNew") != "true" {
		key, err := datastore.DecodeKey(wr.Request.PostForm.Get("templateKey"))
		if err != nil {
			http.Error(wr.ResponseWriter, fmt.Sprintf("Error decoding template key %s: %v", wr.Request.PostForm.Get("templateKey"), err),
				http.StatusBadRequest)
			return
		}
		msg, err := message.Get(ctx, key)
		if err != nil {
			http.Error(wr.ResponseWriter, fmt.Sprintf("Error getting template %s: %v", emailTemplate, err),
				http.StatusInternalServerError)
			return
		}
		emailTemplate = msg.ShortName // Use the existing template name if editing.
		eventKey = msg.Event          // Use the existing event key if editing an existing template.
		keyForSanityCheck = msg.Key() // For sanity check.
	}
	if textBody == "" || htmlBody == "" || subject == "" {
		http.Error(wr.ResponseWriter, "Missing textBody, htmlBody, or subject",
			http.StatusBadRequest)
		return
	}
	decoder := charmap.ISO8859_1.NewDecoder()
	textBody, err := decoder.String(textBody)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Error decoding plaintext for template %s: %v", emailTemplate, err),
			http.StatusInternalServerError)
		return
	}
	htmlBody, err = decoder.String(htmlBody)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Error decoding HTML for template %s: %v", emailTemplate, err),
			http.StatusInternalServerError)
		return
	}
	msg := &message.Message{
		Event:      eventKey,
		ShortName:  emailTemplate,
		Plaintext:  textBody,
		HTML:       htmlBody,
		Subject:    subject,
		Selectable: true,
	}
	if keyForSanityCheck != nil && msg.Key().Encode() != keyForSanityCheck.Encode() {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Key mismatch for template %s: %s != %s", emailTemplate, msg.Key().Encode(), keyForSanityCheck.Encode()),
			http.StatusBadRequest)
		return
	}
	if err := message.SaveTemplate(ctx, msg); err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Error saving template %s: %v", emailTemplate, err),
			http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(wr.ResponseWriter, "Saved template %s", emailTemplate)
}

func handlePreviewMailTemplate(ctx context.Context, wr *WrappedRequest) {

}

func handleViewMyInvitation(ctx context.Context, wr *WrappedRequest) {
	handleMailPage(ctx, wr, "initial_invitation", "viewMyInvitation.html")
}

func handleMailPage(ctx context.Context, wr *WrappedRequest, emailTemplate, htmlTemplate string) {
	// TODO: What data do we send this?
	realizedInvitation := makeRealizedInvitation(ctx, wr.LoginInfo.InvitationKey,
		wr.LoginInfo.Invitation)
	roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event,
		wr.LoginInfo.Invitation, wr.LoginInfo.InvitationKey)
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
		"LoginLink":   MakeLoginUrl(wr.LoginInfo.Person, true),
		"RoomingInfo": roomingInfo,
		"Env":         wr.GetEnvForTemplates(),
		"Unreserved":  unreserved,
	}
	text, html, subject, err := RenderMail(ctx, wr.Event.Key, emailTemplate, emailData, true)
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

func handleDoSendMail(ctx context.Context, wr *WrappedRequest) {
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
			emailData["LoginLink"] = MakeLoginUrl(p, true)
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
		return SendMailViaSendgrid(ctx, wr, emailTemplate, emailData, headerData)
	}
	if err := distributor.Distribute(ctx, wr, senderFunc); err != nil {
		// Email distributors output info as they go, so don't issue an HTTP error.
		fmt.Fprintf(wr.ResponseWriter, "Error from email distributor: %v", err)
	}
}

func handleListMail(ctx context.Context, wr *WrappedRequest) {
	templates, err := message.ListTemplates(ctx, wr.Event.Key)
	if err != nil {
		log.Printf("Error listing templates: %v", err)
		fmt.Fprintf(wr.ResponseWriter, "Error listing templates: %v", err)
		return
	}

	functionMap := template.FuncMap{
		"eventName": func(eventKey *datastore.Key) string {
			if eventKey == nil {
				return "global"
			}
			event, err := event.GetEvent(ctx, eventKey)
			if err != nil {
				log.Printf("Error getting event for key %v: %v", eventKey, err)
				return "Error getting event"
			}
			return event.ShortName
		},
		"makeSendMailLink": func(templateInfo message.MessageInfo) string {
			return "/sendMail?emailTemplate=" + templateInfo.Key.Encode()
		},
		"makeEditMailLink": func(templateInfo message.MessageInfo) string {
			return "/editMail?emailTemplate=" + templateInfo.Key.Encode()
		},
	}
	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/listEmail.html"))
	data := wr.MakeTemplateData(map[string]any{"Templates": templates})
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "listEmail.html", data); err != nil {
		log.Println(err)
	}
}

const senders = "Dana Scott and Chris Shabsin"

type SendgridEnvironment struct {
	EmailClient *sendgrid.Client

	EventKey      *datastore.Key
	BccAddress    string
	SenderAddress string
}

func SendMailViaSendgrid(ctx context.Context, wr *WrappedRequest, templatePrefix string, data any,
	headerData MailHeaderInfo) error {
	sg := &SendgridEnvironment{
		EmailClient:   wr.GetEmailClient(),
		EventKey:      wr.Event.Key,
		BccAddress:    wr.GetBccAddress(),
		SenderAddress: wr.GetSenderAddress(),
	}
	return SendMailViaSendgridImpl(ctx, sg, templatePrefix, data, headerData)
}
func SendMailViaSendgridImpl(ctx context.Context, sg *SendgridEnvironment, templatePrefix string, data any,
	headerData MailHeaderInfo) error {
	text, html, subject, err := RenderMail(ctx, sg.EventKey, templatePrefix, data,
		/* needSubject = */ headerData.Subject == "")
	if headerData.Subject != "" {
		subject = headerData.Subject
	}
	if err != nil {
		log.Printf("Error rendering mail: %v", err)
		return err
	}
	personalizations := []*mail.Personalization{
		ToListPersonalization(headerData.To),
	}

	if headerData.BccSelf {
		bccPers := mail.NewPersonalization()
		bccPers.AddBCCs(mail.NewEmail("", sg.BccAddress))
		personalizations = append(personalizations, bccPers)
	}

	// TODO(cshabsin): get string name from somewhere environmental?
	message := &mail.SGMailV3{
		From:    mail.NewEmail(senders, sg.SenderAddress),
		Subject: subject,
		Content: []*mail.Content{
			mail.NewContent("text/plain", text),
			mail.NewContent("text/html", html),
		},
		Personalizations: personalizations,
	}

	log.Printf("sending mail to %v: %v", headerData.To, message)
	if resp, err := sg.EmailClient.Send(message); err != nil {
		log.Printf("sendgrid.Send got err: %v, %v", resp, err)
	} else {
		log.Printf("sendgrid.Send got resp: %v", resp)
	}
	return nil
}

func ToListPersonalization(to []string) *mail.Personalization {
	mailPersonalizations := mail.NewPersonalization()
	for _, to := range to {
		mailPersonalizations.AddTos(mail.NewEmail("", to))
	}
	return mailPersonalizations
}

func ToPersonalization(name, addr string) *mail.Personalization {
	mailPersonalizations := mail.NewPersonalization()
	mailPersonalizations.AddTos(mail.NewEmail(name, addr))
	return mailPersonalizations
}

func sendErrorMail(wr *WrappedRequest, message string) {
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
