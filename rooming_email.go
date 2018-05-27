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

func handleTestSendRoomingEmail(wr WrappedRequest) {
	rendered_mail, err := getRoomingEmails(wr)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	for _, rm := range rendered_mail {
		wr.ResponseWriter.Write([]byte(rm.HTML))
	}
}

func handleAskSendRoomingEmail(wr WrappedRequest) {
	rendered_mail, err := getRoomingEmails(wr)
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

func handleSendTestRoomingEmail(wr WrappedRequest) {
	handleSendRoomingEmail(wr, true)
}

func handleSendRealRoomingEmail(wr WrappedRequest) {
	handleSendRoomingEmail(wr, false)
}

func handleSendRoomingEmail(wr WrappedRequest, isTest bool) {
	if wr.Method != "POST" {
		http.Error(wr.ResponseWriter, "Invalid GET on send mail handler.",
			http.StatusBadRequest)
		return
	}
	rendered_mail, err := getRoomingEmails(wr)
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

func getRoomingEmails(wr WrappedRequest) (map[int64]RenderedMail, error) {
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

	buildingsMap := getBuildingMapForVenue(wr.Context, wr.Event.Venue)
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
		log.Errorf(ctx, "%v, %v needs double bed: %v", *building, *room, doubleBedNeeded)
		log.Errorf(ctx, "shareBedBit: %v, building.Properties: %v, room.Properties: %v", shareBedBit, building.Properties, room.Properties)
		if doubleBedNeeded && (((building.Properties | room.Properties) & shareBedBit) == shareBedBit) {
			log.Errorf(ctx, "Trying the beds")
			for _, bed := range room.Beds {
				if bed == Double || bed == Queen || bed == King {
					log.Errorf(ctx, "Found bed: %v", bed)
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
	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/PSR2018/email/rooming.html"))

	textFunctionMap := text_template.FuncMap{
		"HasHousingPreference":        RealInvHasHousingPreference,
		"PronounString":               GetPronouns,
		"CollectiveAddressFirstNames": CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
		"DerefPeople":                 DerefPeople,
	}
	text_tpl := text_template.Must(text_template.New("").Funcs(textFunctionMap).ParseGlob("templates/PSR2018/email/rooming.html"))

	rendered_mail := make(map[int64]RenderedMail, 0)
	for invitation, bookings := range allInviteeBookings {
		// invitation is ID from key.
		ri := makeRealizedInvitation(ctx, *datastore.NewKey(ctx, "Invitation", "", invitation, nil), *invitationMap[invitation])
		people_coming := make([]Person, 0)
		for i, p := range ri.Invitees {
			if ri.RsvpMap[p.Key].Attending {
				people_coming = append(people_coming, ri.InviteePeople[i])
			}
		}
		unreserved := make([]BuildingRoom, 0)
		for _, booking := range bookings {
			if !booking.ReservationMade {
				unreserved = append(unreserved, BuildingRoom{booking.Room, booking.Building})
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
				"PeopleComing":    people_coming,
				"Unreserved":      unreserved,
			})
			var text bytes.Buffer
			if err := text_tpl.ExecuteTemplate(&text, "rooming_text", data); err != nil {
				log.Errorf(wr.Context, "%v", err)
			}

			var htmlBuf bytes.Buffer
			if err := tpl.ExecuteTemplate(&htmlBuf, "rooming_html", data); err != nil {
				log.Errorf(wr.Context, "%v", err)
			}

			var subject bytes.Buffer
			if err := text_tpl.ExecuteTemplate(&subject, "rooming_subject", data); err != nil {
				log.Errorf(wr.Context, "%v", err)
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
