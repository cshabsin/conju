package conju

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"cloud.google.com/go/datastore"
	"github.com/cshabsin/conju/conju/mailer"
	"github.com/cshabsin/conju/model/event"
	"github.com/cshabsin/conju/model/message"
	"github.com/cshabsin/conju/model/person"
	"github.com/gin-gonic/gin"
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

func handleSendMail(c *gin.Context) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	wr.Request.ParseForm()
	emailTemplates, ok := wr.Request.Form["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		emailTemplates, ok = wr.Request.PostForm["emailTemplate"]
	}
	if !ok || len(emailTemplates) == 0 {
		handleListMail(c)
		return
	}
	key, err := datastore.DecodeKey(emailTemplates[0])
	if err != nil {
		c.String(http.StatusBadRequest, "Error decoding email template key %s: %v", emailTemplates[0], err)
		return
	}
	msg, err := message.Get(ctx, key)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error getting email template %s: %v", emailTemplates[0], err)
		return
	}
	handleMailPage(c, msg.ShortName, "sendEmail.html")
}

func handleEditMail(c *gin.Context) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
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
			handleListMail(c)
			return
		}
		templateKey = emailTemplates[0]
		key, err := datastore.DecodeKey(templateKey)
		if err != nil {
			c.String(http.StatusBadRequest, "Error decoding email template key %s: %v", templateKey, err)
			return
		}
		msg, err := message.Get(ctx, key)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error getting template %s: %v", templateKey, err)
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
	c.Header("Content-Type", "text/html")
	if err := webTpl.ExecuteTemplate(c.Writer, "editEmail.html", data); err != nil {
		c.String(http.StatusInternalServerError, "Error rendering edit mail page: %v", err)
		return
	}
}

func handleSaveMail(c *gin.Context) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	wr.Request.ParseForm()
	emailTemplates, ok := wr.Request.PostForm["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		c.String(http.StatusBadRequest, "%s issued without emailTemplate?", c.Request.URL.Path)
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
			c.String(http.StatusBadRequest, "Error decoding template key %s: %v", wr.Request.PostForm.Get("templateKey"), err)
			return
		}
		msg, err := message.Get(ctx, key)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error getting template %s: %v", emailTemplate, err)
			return
		}
		emailTemplate = msg.ShortName // Use the existing template name if editing.
		eventKey = msg.Event          // Use the existing event key if editing an existing template.
		keyForSanityCheck = msg.Key() // For sanity check.
	}
	if textBody == "" || htmlBody == "" || subject == "" {
		c.String(http.StatusBadRequest, "Missing textBody, htmlBody, or subject")
		return
	}
	decoder := charmap.ISO8859_1.NewDecoder()
	textBody, err := decoder.String(textBody)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error decoding plaintext for template %s: %v", emailTemplate, err)
		return
	}
	htmlBody, err = decoder.String(htmlBody)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error decoding HTML for template %s: %v", emailTemplate, err)
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
		c.String(http.StatusBadRequest, "Key mismatch for template %s: %s != %s", emailTemplate, msg.Key().Encode(), keyForSanityCheck.Encode())
		return
	}
	if err := message.SaveTemplate(ctx, msg); err != nil {
		c.String(http.StatusInternalServerError, "Error saving template %s: %v", emailTemplate, err)
		return
	}
	c.String(http.StatusOK, "Saved template %s", emailTemplate)
}

func handlePreviewMailTemplate(c *gin.Context) {

}

func handleViewMyInvitation(c *gin.Context) {
	handleMailPage(c, "initial_invitation", "viewMyInvitation.html")
}

func handleMailPage(c *gin.Context, emailTemplate, htmlTemplate string) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
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
		c.String(http.StatusInternalServerError, "Rendering mail: %v", err)
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
		c.String(http.StatusInternalServerError, "Parsing files: %v", err)
		return
	}
	if err := tpl.ExecuteTemplate(c.Writer, htmlTemplate, data); err != nil {
		c.String(http.StatusInternalServerError, "Rendering HTML display: %v", err)
		return
	}
}

func handleDoSendMail(c *gin.Context) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	wr.Request.ParseForm()
	emailTemplates, ok := wr.Request.PostForm["emailTemplate"]
	if !ok || len(emailTemplates) == 0 {
		c.String(http.StatusBadRequest, "%s issued without emailTemplate?", c.Request.URL.Path)
		return
	}
	emailTemplate := emailTemplates[0]
	distributors, ok := wr.Request.PostForm["distributor"]
	if !ok || len(distributors) == 0 {
		c.String(http.StatusBadRequest, "%s issued without distributor?", c.Request.URL.Path)
		return
	}
	distributorName := distributors[0]
	distributor, ok := AllDistributors[distributorName]
	if !ok {
		c.String(http.StatusBadRequest, "Bad distributor name: %s", distributorName)
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
		return SendMail(c, emailTemplate, emailData, headerData)
	}
	if err := distributor.Distribute(c, senderFunc); err != nil {
		// Email distributors output info as they go, so don't issue an HTTP error.
		c.String(http.StatusInternalServerError, "Error from email distributor: %v", err)
	}
}

func handleListMail(c *gin.Context) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	templates, err := message.ListTemplates(ctx, wr.Event.Key)
	if err != nil {
		log.Printf("Error listing templates: %v", err)
		c.String(http.StatusInternalServerError, "Error listing templates: %v", err)
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
	if err := tpl.ExecuteTemplate(c.Writer, "listEmail.html", data); err != nil {
		log.Println(err)
	}
}

const senders = "Dana Scott and Chris Shabsin"

type EmailData struct {
	EmailClient *sendgrid.Client

	EventKey      *datastore.Key
	BccAddress    string
	SenderAddress string
}

func SendMail(c *gin.Context, templatePrefix string, data any,
	headerData MailHeaderInfo) error {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	sg := &EmailData{
		EmailClient:   wr.GetEmailClient(),
		EventKey:      wr.Event.Key,
		BccAddress:    wr.GetBccAddress(),
		SenderAddress: wr.GetSenderAddress(),
	}
	return SendMailImpl(ctx, sg, templatePrefix, data, headerData)
}

func SendMailImpl(ctx context.Context, sg *EmailData, templatePrefix string, data any,
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

	mailClient, err := mailer.FromContext(ctx)
	if err != nil {
		log.Printf("Error getting mail client: %v", err)
		return err
	}
	msg := &mailer.Message{
		To: headerData.To,
		From: mailer.Email{
			Name: senders,
			Addr: sg.SenderAddress,
		},
		Subject: subject,
		Text:    text,
		HTML:    html,
	}

	if headerData.BccSelf {
		msg.BCC = append(msg.BCC, mailer.Email{
			Addr: sg.BccAddress,
		})
	}

	log.Printf("sending mail to %v", headerData.To)
	if err := mailClient.Send(ctx, msg); err != nil {
		log.Printf("mailclient.Send got err: %v", err)
		return err
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