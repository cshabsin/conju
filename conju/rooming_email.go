package conju

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	text_template "text/template"

	"gopkg.in/sendgrid/sendgrid-go.v2"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type RenderedMail struct {
	Person  Person
	Text    string
	HTML    string
	Subject string
}

func handleTestSendUpdatesEmail(wr WrappedRequest) {
	handleTestSendRoomingRelatedEmail(wr, "updates")
}

func handleTestSendRoomingEmail(wr WrappedRequest) {
	handleTestSendRoomingRelatedEmail(wr, "rooming")
}

func handleTestSendFinalEmail(wr WrappedRequest) {
	handleTestSendRoomingRelatedEmail(wr, "final")
}

func handleTestSendRoomingRelatedEmail(wr WrappedRequest, emailName string) {
	rendered_mail, err := getRoomingEmails(wr, emailName)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for _, rm := range rendered_mail {
		wr.ResponseWriter.Write([]byte(rm.Text))
	}
}

func handleAskSendRoomingEmail(wr WrappedRequest) {
	rendered_mail, err := getRoomingEmails(wr, "rooming")
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(wr.ResponseWriter, `
	Number of emails to send: %d<p>
	<form method="POST" action="/doSendTestRoomingEmail">
	<input type="submit" value="Send Test Mail">
	</form>
	<form method="POST" action="/doSendRealRoomingEmail">
	<input type="submit" value="Send Real Mail">
	</form>
`, len(rendered_mail))
}

func handleAskSendUpdatesEmail(wr WrappedRequest) {
	rendered_mail, err := getRoomingEmails(wr, "updates")
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(wr.ResponseWriter, `
	Number of emails to send: %d<p>
	<form method="POST" action="/doSendTestUpdatesEmail">
	<input type="submit" value="Send Test Mail">
	</form>
	<form method="POST" action="/doSendRealUpdatesEmail">
	<input type="submit" value="Send Real Mail">
	</form>
`, len(rendered_mail))
}

func handleSendTestRoomingEmail(wr WrappedRequest) {
	handleSendRoomingEmail(wr, "rooming", true)
}

func handleSendRealRoomingEmail(wr WrappedRequest) {
	handleSendRoomingEmail(wr, "rooming", false)
}

func handleSendTestUpdatesEmail(wr WrappedRequest) {
	handleSendRoomingEmail(wr, "updates", true)
}

func handleSendRealUpdatesEmail(wr WrappedRequest) {
	handleSendRoomingEmail(wr, "updates", false)
}

func handleSendRoomingEmail(wr WrappedRequest, emailName string, isTest bool) {
	if wr.Method != "POST" {
		http.Error(wr.ResponseWriter, "Invalid GET on send mail handler.",
			http.StatusBadRequest)
		return
	}
	rendered_mail, err := getRoomingEmails(wr, emailName)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	for _, to_render := range rendered_mail {
		p := to_render.Person
		message := sendgrid.NewMail()
		if isTest {
			message.AddTo(fmt.Sprintf("%s test <%s>", p.FullName(),
				wr.GetBccAddress()))
		} else {
			message.AddTo(fmt.Sprintf("%s <%s>", p.FullName(), p.Email))
			message.AddBcc(wr.GetBccAddress())
		}
		message.SetSubject(to_render.Subject)
		message.SetHTML(to_render.HTML)
		message.SetText(to_render.Text)
		message.SetFrom(wr.GetSenderAddress())
		fmt.Fprintf(wr.ResponseWriter, "Sending to %s (isTest = %v)<p>", p.FullName(), isTest)
		err = wr.GetEmailClient().Send(message)
		if err != nil {
			log.Errorf(wr.Context, "Error sending mail: %v", err)
		}
	}
}

func getRoomingEmails(wr WrappedRequest, emailName string) (map[int64]RenderedMail, error) {
	// Cribbed heavily from handleRoomingReport
	ctx := wr.Context

	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	_, err := q.GetAll(ctx, &bookings)
	if err != nil {
		log.Errorf(ctx, "fetching bookings: %v", err)
	}

	var rooms = make([]*Room, len(wr.Event.Rooms))
	err = datastore.GetMulti(ctx, wr.Event.Rooms, rooms)
	if err != nil {
		log.Errorf(ctx, "fetching rooms: %v", err)
	}

	// Map room ID -> Room
	roomsMap := make(map[int64]*Room)
	for i, room := range rooms {
		roomsMap[wr.Event.Rooms[i].IntID()] = room
	}

	var peopleToLookUp []*datastore.Key
	for _, booking := range bookings {
		peopleToLookUp = append(peopleToLookUp, booking.Roommates...)
	}

	personMap := make(map[int64]*Person)
	var people = make([]*Person, len(peopleToLookUp))
	err = datastore.GetMulti(ctx, peopleToLookUp, people)
	if err != nil {
		log.Errorf(ctx, "fetching people: %v", err)
	}

	for i, person := range people {
		personMap[peopleToLookUp[i].IntID()] = person
	}

	var invitations []*Invitation
	q = datastore.NewQuery("Invitation").Filter("Event =", wr.EventKey)
	invitationKeys, err := q.GetAll(ctx, &invitations)
	if err != nil {
		log.Errorf(ctx, "fetching invitations: %v", err)
	}

	personToInvitationMap := make(map[int64]int64)
	invitationMap := make(map[int64]*Invitation)
	for i, inv := range invitations {
		invitationMap[invitationKeys[i].IntID()] = inv
		for _, person := range inv.Invitees {
			personToInvitationMap[person.IntID()] = invitationKeys[i].IntID()
		}
	}
	shareBedBit := GetAllHousingPreferenceBooleans()[ShareBed].Bit

	type BuildingRoom struct {
		Room     *Room
		Building *Building
	}
	type InviteeRoomBookings struct {
		Building            *Building
		Room                *Room
		Roommates           []*Person // People from this invitation.
		RoomSharers         []*Person // People from outside the invitation.
		ShowConvertToDouble bool
		ReservationMade     bool
	}
	type InviteeBookings map[BuildingRoom]InviteeRoomBookings

	buildingsMap := getBuildingMapForVenue(ctx, wr.Event.Venue)
	allInviteeBookings := make(map[int64]InviteeBookings)
	for _, booking := range bookings {
		room := roomsMap[booking.Room.IntID()]
		buildingId := booking.Room.Parent().IntID()
		building := buildingsMap[buildingId]
		buildingRoom := BuildingRoom{room, building}

		// Figure out if anyone's invitation signals need for a double bed.
		doubleBedNeeded := false
		for _, person := range booking.Roommates {
			invitation := invitationMap[personToInvitationMap[person.IntID()]]
			doubleBedNeeded = doubleBedNeeded || (invitation.HousingPreferenceBooleans&shareBedBit == shareBedBit)
		}

		// Figure out if we need them to tell PSR to convert twin beds to double.
		showConvertToDouble := doubleBedNeeded

		if doubleBedNeeded && (((building.Properties | room.Properties) & shareBedBit) == shareBedBit) {
			for _, bed := range room.Beds {
				if bed == Double || bed == Queen || bed == King {
					showConvertToDouble = false
					break
				}
			}
		}

		for _, person := range booking.Roommates {
			invitation := personToInvitationMap[person.IntID()]

			inviteeBookings, found := allInviteeBookings[invitation]
			if !found {
				inviteeBookings = make(InviteeBookings)
				allInviteeBookings[invitation] = inviteeBookings
			}
			_, found = inviteeBookings[buildingRoom]
			if !found {
				roommates := make([]*Person, 0)
				roomSharers := make([]*Person, 0)
				for _, maybeRoommate := range booking.Roommates {
					maybeRoommatePerson := personMap[maybeRoommate.IntID()]
					if personToInvitationMap[maybeRoommate.IntID()] == invitation {
						roommates = append(roommates, maybeRoommatePerson)
					} else {
						roomSharers = append(roomSharers, maybeRoommatePerson)
					}
				}
				inviteeBookings[buildingRoom] = InviteeRoomBookings{
					Building:            building,
					Room:                room,
					Roommates:           roommates,
					RoomSharers:         roomSharers,
					ShowConvertToDouble: showConvertToDouble,
					ReservationMade:     booking.Reserved,
				}
			}
		}
	}

	functionMap := template.FuncMap{
		"HasHousingPreference":        RealInvHasHousingPreference,
		"PronounString":               GetPronouns,
		"CollectiveAddressFirstNames": CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
		"DerefPeople":                 DerefPeople,
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/PSR2018/email/" + emailName + ".html"))

	textFunctionMap := text_template.FuncMap{
		"HasHousingPreference":        RealInvHasHousingPreference,
		"PronounString":               GetPronouns,
		"CollectiveAddressFirstNames": CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
		"DerefPeople":                 DerefPeople,
	}
	text_tpl := text_template.Must(text_template.New("").Funcs(textFunctionMap).ParseGlob("templates/PSR2018/email/" + emailName + ".html"))

	rendered_mail := make(map[int64]RenderedMail, 0)
	for invitation, bookings := range allInviteeBookings {
		// invitation is ID from key.
		ri := makeRealizedInvitation(ctx, datastore.NewKey(ctx, "Invitation", "", invitation, nil), invitationMap[invitation])
		unreserved := make([]BuildingRoom, 0)
		for _, booking := range bookings {
			if !booking.ReservationMade {
				unreserved = append(unreserved, BuildingRoom{booking.Room, booking.Building})
			}
		}

		thursday := false
		for i := range ri.InviteePeople {
			status := ri.RsvpMap[ri.Invitees[i].Key].Status
			if status == ThuFriSat {
				thursday = true
				break
			}
		}

		for i, p := range ri.InviteePeople {
			if p.Email == "" {
				continue
			}
			if !ri.RsvpMap[ri.Invitees[i].Key].Attending {
				continue
			}
			data := wr.MakeTemplateData(map[string]interface{}{
				"Invitation":      ri,
				"InviteeBookings": bookings,
				"LoginLink":       makeLoginUrl(&p),
				"PeopleComing":    ri.GetPeopleComing(),
				"Thursday":        thursday,
				"Unreserved":      unreserved,
			})
			var text bytes.Buffer
			if err := text_tpl.ExecuteTemplate(&text, emailName+"_text", data); err != nil {
				log.Errorf(ctx, "%v", err)
			}

			var htmlBuf bytes.Buffer
			if err := tpl.ExecuteTemplate(&htmlBuf, emailName+"_html", data); err != nil {
				log.Errorf(ctx, "%v", err)
			}

			var subject bytes.Buffer
			if err := text_tpl.ExecuteTemplate(&subject, emailName+"_subject", data); err != nil {
				log.Errorf(ctx, "%v", err)
			}
			rendered_mail[p.DatastoreKey.IntID()] = RenderedMail{p, text.String(), htmlBuf.String(), subject.String()}
		}
	}
	return rendered_mail, nil
}

func MakeSharerName(p *Person) string {
	s := p.FullName()
	if p.Email != "" {
		s = s + " (" + p.Email + ")"
	}
	return s
}

func DerefPeople(people []*Person) []Person {
	dp := make([]Person, len(people))
	for i, p := range people {
		dp[i] = *p
	}
	return dp
}
