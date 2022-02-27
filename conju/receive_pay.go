package conju

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/cshabsin/conju/model/person"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

func handleReceivePay(wr WrappedRequest) {
	wr.Request.ParseForm()
	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, err := datastore.DecodeKey(invitationKeyEncoded)
	if err != nil {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("Error decoding invitation key: %v", err),
			http.StatusBadRequest)
	}

	var invitation Invitation
	err = datastore.Get(wr.Context, invitationKey, &invitation)
	if err != nil {
		log.Errorf(wr.Context, "error getting invitation: %v", err)
	}

	realizedInvitation := makeRealizedInvitation(wr.Context, invitationKey, &invitation)
	roomingInfo := getRoomingInfoWithInvitation(wr, &invitation, invitationKey)
	data := wr.MakeTemplateData(map[string]interface{}{
		"Invitation":  realizedInvitation,
		"RoomingInfo": roomingInfo,
	})

	functionMap := template.FuncMap{
		"PronounString":               person.GetPronouns,
		"HasPreference":               HasPreference,
		"DerefPeople":                 DerefPeople,
		"CollectiveAddressFirstNames": person.CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/receive_pay.html", "templates/roomingInfo.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "receive_pay.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}

func handleDoReceivePay(wr WrappedRequest) {
	wr.Request.ParseForm()

	payStr := wr.Request.Form.Get("pay")
	pay, err := strconv.ParseFloat(payStr, 64)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Error retrieving pay from form: %v", err), http.StatusBadRequest)
		return
	}
	payDateStr := wr.Request.Form.Get("pay_date")
	payDate, err := time.Parse("2006-01-02", payDateStr)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Invalid date string from form: %v", err), http.StatusBadRequest)
		return
	}

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, err := datastore.DecodeKey(invitationKeyEncoded)
	if err != nil {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("Error decoding invitation key: %v", err),
			http.StatusBadRequest)
	}

	var invitation Invitation
	err = datastore.Get(wr.Context, invitationKey, &invitation)
	if err != nil {
		log.Errorf(wr.Context, "error getting invitation: %v", err)
	}
	invitation.ReceivedPay = float64(pay)
	invitation.ReceivedPayDate = payDate
	invitation.ReceivedPayMethod = wr.Request.Form.Get("pay_method")
	_, err = datastore.Put(wr.Context, invitationKey, &invitation)
	if err != nil {
		log.Errorf(wr.Context, "error saving invitation: %v", err)
	}
	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}
