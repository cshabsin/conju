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
	"github.com/gin-gonic/gin"

	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/model/person"
)

func handleReceivePay(c *gin.Context) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	wr.Request.ParseForm()
	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, err := datastore.DecodeKey(invitationKeyEncoded)
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Error decoding invitation key: %v", err))
	}

	var invitation Invitation
	err = dsclient.FromContext(ctx).Get(ctx, invitationKey, &invitation)
	if err != nil {
		log.Printf("error getting invitation: %v", err)
	}

	realizedInvitation := makeRealizedInvitation(ctx, invitationKey, &invitation)
	roomingInfo := GetRoomingInfoWithInvitation(ctx, wr.GetBookingInfo(ctx), wr.Event, &invitation, invitationKey)
	data := wr.MakeTemplateData(map[string]any{
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
	if err := tpl.ExecuteTemplate(c.Writer, "receive_pay.html", data); err != nil {
		log.Printf("%v", err)
	}
}

func handleDoReceivePay(c *gin.Context) {
	wr, _ := c.MustGet("wrappedRequest").(*WrappedRequest)
	ctx := c.Request.Context()
	wr.Request.ParseForm()

	payStr := wr.Request.Form.Get("pay")
	pay, err := strconv.ParseFloat(payStr, 64)
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Error retrieving pay from form: %v", err))
		return
	}
	payDateStr := wr.Request.Form.Get("pay_date")
	payDate, err := time.Parse("2006-01-02", payDateStr)
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Invalid date string from form: %v", err))
		return
	}

	invitationKeyEncoded := wr.Request.Form.Get("invitation")
	invitationKey, err := datastore.DecodeKey(invitationKeyEncoded)
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Error decoding invitation key: %v", err))
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
	c.Redirect(http.StatusSeeOther, "invitations")
}