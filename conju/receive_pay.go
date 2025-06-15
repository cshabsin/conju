package conju

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/model/person"
)

func handleReceivePay(ctx context.Context, wr WrappedRequest) {
	wr.Request.ParseForm()
	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, err := datastore.DecodeKey(invitationKeyEncoded)
	if err != nil {
		http.Error(wr.ResponseWriter,
			fmt.Sprintf("Error decoding invitation key: %v", err),
			http.StatusBadRequest)
	}

	var invitation Invitation
	err = dsclient.FromContext(ctx).Get(ctx, invitationKey, &invitation)
	if err != nil {
		log.Printf("error getting invitation: %v", err)
	}

	realizedInvitation := makeRealizedInvitation(ctx, invitationKey, &invitation)
	roomingInfo := getRoomingInfoWithInvitation(ctx, wr, &invitation, invitationKey)
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
		log.Printf("%v", err)
	}
}

func handleDoReceivePay(ctx context.Context, wr WrappedRequest) {
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
	err = dsclient.FromContext(ctx).Get(ctx, invitationKey, &invitation)
	if err != nil {
		log.Printf("error getting invitation: %v", err)
	}
	invitation.ReceivedPay = float64(pay)
	invitation.ReceivedPayDate = payDate
	invitation.ReceivedPayMethod = wr.Request.Form.Get("pay_method")
	_, err = dsclient.FromContext(ctx).Put(ctx, invitationKey, &invitation)
	if err != nil {
		log.Printf("error saving invitation: %v", err)
	}
	http.Redirect(wr.ResponseWriter, wr.Request, "invitations", http.StatusSeeOther)
}
