package conju

import (
	"fmt"
	"html/template"
	"net/http"

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
		"PronounString":               GetPronouns,
		"HasPreference":               HasPreference,
		"DerefPeople":                 DerefPeople,
		"CollectiveAddressFirstNames": CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/main.html", "templates/receive_pay.html", "templates/roomingInfo.html"))
	if err := tpl.ExecuteTemplate(wr.ResponseWriter, "receive_pay.html", data); err != nil {
		log.Errorf(wr.Context, "%v", err)
	}
}
