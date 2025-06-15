package conju

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	text_template "text/template"

	"cloud.google.com/go/datastore"

	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/housing"
	"github.com/cshabsin/conju/model/person"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type RenderedMail struct {
	Person  person.Person
	Text    string
	HTML    string
	Subject string
}

func handleTestSendUpdatesEmail(ctx context.Context, wr WrappedRequest) {
	handleTestSendRoomingRelatedEmail(ctx, wr, "updates")
}

func handleTestSendRoomingEmail(ctx context.Context, wr WrappedRequest) {
	handleTestSendRoomingRelatedEmail(ctx, wr, "rooming")
}

func handleTestSendFinalEmail(ctx context.Context, wr WrappedRequest) {
	handleTestSendRoomingRelatedEmail(ctx, wr, "final")
}

func handleTestSendRoomingRelatedEmail(ctx context.Context, wr WrappedRequest, emailName string) {
	rendered_mail, err := getRoomingEmails(ctx, wr, emailName)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	for _, rm := range rendered_mail {
		wr.ResponseWriter.Write([]byte(rm.Text))
	}
}

func handleAskSendRoomingEmail(ctx context.Context, wr WrappedRequest) {
	rendered_mail, err := getRoomingEmails(ctx, wr, "rooming")
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

func handleAskSendUpdatesEmail(ctx context.Context, wr WrappedRequest) {
	rendered_mail, err := getRoomingEmails(ctx, wr, "updates")
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

func handleSendTestRoomingEmail(ctx context.Context, wr WrappedRequest) {
	handleSendRoomingEmail(ctx, wr, "rooming", true)
}

func handleSendRealRoomingEmail(ctx context.Context, wr WrappedRequest) {
	handleSendRoomingEmail(ctx, wr, "rooming", false)
}

func handleSendTestUpdatesEmail(ctx context.Context, wr WrappedRequest) {
	handleSendRoomingEmail(ctx, wr, "updates", true)
}

func handleSendRealUpdatesEmail(ctx context.Context, wr WrappedRequest) {
	handleSendRoomingEmail(ctx, wr, "updates", false)
}

func handleSendRoomingEmail(ctx context.Context, wr WrappedRequest, emailName string, isTest bool) {
	if wr.Method != "POST" {
		http.Error(wr.ResponseWriter, "Invalid GET on send mail handler.",
			http.StatusBadRequest)
		return
	}
	rendered_mail, err := getRoomingEmails(ctx, wr, emailName)
	if err != nil {
		http.Error(wr.ResponseWriter, fmt.Sprintf("Rendering mail: %v", err),
			http.StatusInternalServerError)
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	for _, to_render := range rendered_mail {
		p := to_render.Person
		message := &mail.SGMailV3{
			From:    mail.NewEmail(senders, wr.GetSenderAddress()),
			Subject: to_render.Subject,
			Content: []*mail.Content{
				mail.NewContent("text/plain", to_render.Text),
				mail.NewContent("text/html", to_render.HTML),
			},
		}
		if isTest {
			message.AddPersonalizations(ToPersonalization(fmt.Sprintf("%s test", p.FullName()), wr.GetBccAddress()))
		} else {
			p := ToPersonalization(p.FullName(), p.Email)
			p.AddBCCs(mail.NewEmail("", wr.GetBccAddress()))
			message.AddPersonalizations(p)
		}
		fmt.Fprintf(wr.ResponseWriter, "Sending to %s (isTest = %v)<p>", p.FullName(), isTest)
		_, err = wr.GetEmailClient().Send(message)
		if err != nil {
			log.Printf("Error sending mail: %v", err)
		}
	}
}

func getRoomingEmails(ctx context.Context, wr WrappedRequest, emailName string) (map[int64]RenderedMail, error) {
	// Cribbed heavily from handleRoomingReport
	var bookings []Booking
	q := datastore.NewQuery("Booking").Ancestor(wr.EventKey)
	_, err := dsclient.FromContext(ctx).GetAll(ctx, q, &bookings)
	if err != nil {
		log.Printf("fetching bookings: %v", err)
	}

	var rooms = make([]*housing.Room, len(wr.Event.Rooms))
	err = dsclient.FromContext(ctx).GetMulti(ctx, wr.Event.Rooms, rooms)
	if err != nil {
		log.Printf("fetching rooms: %v", err)
	}

	// Map room ID -> Room
	roomsMap := make(map[int64]*housing.Room)
	for i, room := range rooms {
		roomsMap[wr.Event.Rooms[i].ID] = room
	}

	var peopleToLookUp []*datastore.Key
	for _, booking := range bookings {
		peopleToLookUp = append(peopleToLookUp, booking.Roommates...)
	}

	personMap := make(map[int64]*person.Person)
	var people = make([]*person.Person, len(peopleToLookUp))
	err = dsclient.FromContext(ctx).GetMulti(ctx, peopleToLookUp, people)
	if err != nil {
		log.Printf("fetching people: %v", err)
	}

	for i, person := range people {
		personMap[peopleToLookUp[i].ID] = person
	}

	var invitations []*Invitation
	q = datastore.NewQuery("Invitation").FilterField("Event", "=", wr.EventKey)
	invitationKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &invitations)
	if err != nil {
		log.Printf("fetching invitations: %v", err)
	}

	personToInvitationMap := make(map[int64]int64)
	invitationMap := make(map[int64]*Invitation)
	for i, inv := range invitations {
		invitationMap[invitationKeys[i].ID] = inv
		for _, person := range inv.Invitees {
			personToInvitationMap[person.ID] = invitationKeys[i].ID
		}
	}
	shareBedBit := GetAllHousingPreferenceBooleans()[ShareBed].Bit

	type BuildingRoom struct {
		Room     *housing.Room
		Building *housing.Building
	}
	type InviteeRoomBookings struct {
		Building            *housing.Building
		Room                *housing.Room
		Roommates           []*person.Person // People from this invitation.
		RoomSharers         []*person.Person // People from outside the invitation.
		ShowConvertToDouble bool
		ReservationMade     bool
	}
	type InviteeBookings map[BuildingRoom]InviteeRoomBookings

	wr.Event.LoadVenue(ctx)
	buildingsMap := getBuildingMapForVenue(ctx, wr.Event.Venue.Key)
	allInviteeBookings := make(map[int64]InviteeBookings)
	for _, booking := range bookings {
		room := roomsMap[booking.Room.ID]
		buildingId := booking.Room.Parent.ID
		building := buildingsMap[buildingId]
		buildingRoom := BuildingRoom{room, building}

		// Figure out if anyone's invitation signals need for a double bed.
		doubleBedNeeded := false
		for _, person := range booking.Roommates {
			invitation := invitationMap[personToInvitationMap[person.ID]]
			doubleBedNeeded = doubleBedNeeded || (invitation.HousingPreferenceBooleans&shareBedBit == shareBedBit)
		}

		// Figure out if we need them to tell PSR to convert twin beds to double.
		showConvertToDouble := doubleBedNeeded

		if doubleBedNeeded && (((building.Properties | room.Properties) & shareBedBit) == shareBedBit) {
			for _, bed := range room.Beds {
				if bed == housing.Double || bed == housing.Queen || bed == housing.King {
					showConvertToDouble = false
					break
				}
			}
		}

		for _, per := range booking.Roommates {
			invitation := personToInvitationMap[per.ID]

			inviteeBookings, found := allInviteeBookings[invitation]
			if !found {
				inviteeBookings = make(InviteeBookings)
				allInviteeBookings[invitation] = inviteeBookings
			}
			_, found = inviteeBookings[buildingRoom]
			if !found {
				roommates := make([]*person.Person, 0)
				roomSharers := make([]*person.Person, 0)
				for _, maybeRoommate := range booking.Roommates {
					maybeRoommatePerson := personMap[maybeRoommate.ID]
					if personToInvitationMap[maybeRoommate.ID] == invitation {
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
		"PronounString":               person.GetPronouns,
		"CollectiveAddressFirstNames": person.CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
		"DerefPeople":                 DerefPeople,
	}

	tpl := template.Must(template.New("").Funcs(functionMap).ParseFiles("templates/PSR2022/email/" + emailName + ".html"))

	textFunctionMap := text_template.FuncMap{
		"HasHousingPreference":        RealInvHasHousingPreference,
		"PronounString":               person.GetPronouns,
		"CollectiveAddressFirstNames": person.CollectiveAddressFirstNames,
		"SharerName":                  MakeSharerName,
		"DerefPeople":                 DerefPeople,
	}
	text_tpl := text_template.Must(text_template.New("").Funcs(textFunctionMap).ParseGlob("templates/PSR2022/email/" + emailName + ".html"))

	rendered_mail := make(map[int64]RenderedMail, 0)
	for inv, bookings := range allInviteeBookings {
		// invitation is ID from key.
		ri := makeRealizedInvitation(ctx, datastore.IDKey("Invitation", inv, nil), invitationMap[inv])
		var unreserved []BuildingRoom
		for _, booking := range bookings {
			if !booking.ReservationMade {
				unreserved = append(unreserved, BuildingRoom{booking.Room, booking.Building})
			}
		}

		thursday := false
		for i := range ri.InviteePeople {
			status := ri.RsvpMap[ri.Invitees[i].Key].Status
			if status == invitation.ThuFriSat {
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
				"LoginLink":       makeLoginUrl(&p, true),
				"PeopleComing":    ri.GetPeopleComing(),
				"Thursday":        thursday,
				"Unreserved":      unreserved,
			})
			var text bytes.Buffer
			if err := text_tpl.ExecuteTemplate(&text, emailName+"_text", data); err != nil {
				log.Printf("%v", err)
			}

			var htmlBuf bytes.Buffer
			if err := tpl.ExecuteTemplate(&htmlBuf, emailName+"_html", data); err != nil {
				log.Printf("%v", err)
			}

			var subject bytes.Buffer
			if err := text_tpl.ExecuteTemplate(&subject, emailName+"_subject", data); err != nil {
				log.Printf("%v", err)
			}
			rendered_mail[p.DatastoreKey.ID] = RenderedMail{p, text.String(), htmlBuf.String(), subject.String()}
		}
	}
	return rendered_mail, nil
}

func MakeSharerName(p *person.Person) string {
	s := p.FullName()
	if p.Email != "" {
		s = s + " (" + p.Email + ")"
	}
	return s
}

func DerefPeople(people []*person.Person) []person.Person {
	dp := make([]person.Person, len(people))
	for i, p := range people {
		dp[i] = *p
	}
	return dp
}
