package conju

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/cshabsin/conju/activity"
	"github.com/cshabsin/conju/conju/dsclient"
	"github.com/cshabsin/conju/conju/login"
	"github.com/cshabsin/conju/invitation"
	"github.com/cshabsin/conju/model/event"
	"github.com/cshabsin/conju/model/housing"
	"github.com/cshabsin/conju/model/person"
	"github.com/cshabsin/conju/model/venue"
)

// const Import_Data_Directory = "test_import_data"
const Import_Data_Directory = "real_import_data"

const Guest_Data_File_Name = "Guests_to_Import.tsv"
const RSVP_Data_File_Name = "rsvps.tsv"
const Events_Data_File_Name = "events.tsv"
const Food_File_Name = "food.tsv"
const Activities_File_Name = "activities.tsv"
const Venues_File_Name = "venues.tsv"
const Buildings_File_Name = "buildings.tsv"
const Rooms_File_Name = "rooms.tsv"

func ReloadData(ctx context.Context, wr WrappedRequest) {
	if wr.Method != "POST" {
		http.Error(wr.ResponseWriter, "Invalid GET on reload.",
			http.StatusBadRequest)
		return
	}
	wr.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	ClearAllData(ctx, wr, []string{"Activity", "Event", "CurrentEvent", "Person", "Invitation", "LoginCode", "Venue", "Building", "Room"})
	wr.ResponseWriter.Write([]byte("\n\n"))
	SetupVenues(wr.ResponseWriter, ctx)
	wr.ResponseWriter.Write([]byte("\n\n"))
	time.Sleep(2 * time.Second)
	SetupBuildings(wr.ResponseWriter, ctx)
	wr.ResponseWriter.Write([]byte("\n\n"))
	time.Sleep(2 * time.Second)
	SetupRooms(wr.ResponseWriter, ctx)
	wr.ResponseWriter.Write([]byte("\n\n"))
	time.Sleep(2 * time.Second)
	SetupActivities(wr.ResponseWriter, ctx)
	wr.ResponseWriter.Write([]byte("\n\n"))
	time.Sleep(2 * time.Second)
	SetupEvents(wr.ResponseWriter, ctx)
	wr.ResponseWriter.Write([]byte("\n\n"))
	time.Sleep(2 * time.Second)
	guestMap := ImportGuests(wr.ResponseWriter, ctx)
	wr.ResponseWriter.Write([]byte("\n\n"))
	ImportFoodPreferences(wr.ResponseWriter, ctx, guestMap)
	wr.ResponseWriter.Write([]byte("\n\n"))
	ImportRsvps(wr.ResponseWriter, ctx, guestMap)

}

func SetupActivities(w http.ResponseWriter, ctx context.Context) error {
	activitiesFile, err := os.Open(Import_Data_Directory + "/" + Activities_File_Name)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	defer activitiesFile.Close()

	scanner := bufio.NewScanner(activitiesFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			activityRow := scanner.Text()
			fields := strings.Split(activityRow, "\t")
			keyword := fields[0]
			description := fields[2]
			needsLeader := fields[1] == "TRUE"

			activity := activity.Activity{
				Keyword:     keyword,
				Description: description,
				NeedsLeader: needsLeader,
			}

			_, err := dsclient.FromContext(ctx).Put(ctx, datastore.IncompleteKey("Activity", nil), &activity)
			if err != nil {
				log.Printf("%v", err)
			}
			w.Write([]byte(fmt.Sprintf("Loading activity %s\n", fields[0])))
		}
		processedHeader = true
	}
	return err
}

func AskReloadData(ctx context.Context, wr WrappedRequest) {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	// fmt.Fprintf(wr.ResponseWriter, `
	// <form method="POST" action="/doReloadData">
	// <input type="submit" value="Do it">
	// </form>
	// `)
	fmt.Fprintf(wr.ResponseWriter, "NO")
}

func SetupEvents(w http.ResponseWriter, ctx context.Context) error {
	eventsFile, err := os.Open(Import_Data_Directory + "/" + Events_Data_File_Name)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	defer eventsFile.Close()

	layout := "1/2/2006"

	venuesMap := make(map[string]datastore.Key)
	var venues []venue.Venue
	q := datastore.NewQuery("Venue")
	keys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &venues)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	for i, venueKey := range keys {
		venuesMap[(venues[i]).ShortName] = *venueKey
	}

	buildingsMap := make(map[string]datastore.Key)
	var buildings []housing.Building
	q = datastore.NewQuery("Building")
	keys, err = dsclient.FromContext(ctx).GetAll(ctx, q, &buildings)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	for i, buildingKey := range keys {
		buildingsMap[(buildings[i]).Code] = *buildingKey
	}

	rsvpStatusMap := make(map[string]invitation.RsvpStatus)
	allRsvpStatuses := invitation.GetAllRsvpStatuses()
	for _, status := range allRsvpStatuses {
		rsvpStatusMap[status.ShortDescription] = status.Status
	}

	activityMap := make(map[string]*datastore.Key)
	var activities []activity.Activity
	q = datastore.NewQuery("Activity")
	keys, err = dsclient.FromContext(ctx).GetAll(ctx, q, &activities)
	for i, activityKey := range keys {
		activityMap[(activities[i]).Keyword] = activityKey
	}

	scanner := bufio.NewScanner(eventsFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			eventRow := scanner.Text()

			fields := strings.Split(eventRow, "\t")
			startDate, _ := time.Parse(layout, fields[4])
			endDate, _ := time.Parse(layout, fields[5])
			eventId, _ := strconv.Atoi(fields[0])
			venueKey := venuesMap[fields[3]]
			rsvpStatusStrings := strings.Split(fields[7], ",")
			var rsvpStatuses []invitation.RsvpStatus
			for _, rsvpStatusString := range rsvpStatusStrings {
				rsvpStatuses = append(rsvpStatuses, rsvpStatusMap[rsvpStatusString])
			}

			allActivities := fields[10]
			activities := strings.Split(allActivities, ",")
			var activityKeys []*datastore.Key
			for _, activity := range activities {
				if activity == "" {
					continue
				}
				activityKey := activityMap[activity]
				if activityKey == nil {
					log.Printf("nil activityKey for activity %s", activity)
				}
				//if activityKey != nil {
				activityKeys = append(activityKeys, activityKey)
				//}
			}

			rooms := getRoomsFromString(fields[8], ctx, buildingsMap)

			e := &event.Event{
				EventId:               eventId,
				Name:                  fields[1],
				ShortName:             fields[2],
				StartDate:             startDate,
				EndDate:               endDate,
				RsvpStatuses:          rsvpStatuses,
				InvitationClosingText: fields[9],
				Activities:            activityKeys,
				Current:               fields[6] == "1",
				Rooms:                 rooms,
			}
			e.SetVenueKey(&venueKey)

			err := event.PutEvent(ctx, e)
			if err != nil {
				log.Printf("PutEvent: %v", err)
				w.Write([]byte(fmt.Sprintf("Error calling PutEvent: %v\n", err)))
			}

			w.Write([]byte(fmt.Sprintf("Loading event %s (%s) %s - %s\n", fields[1], fields[2], startDate.Format("01/02/2006"), endDate.Format("01/02/2006"))))
		}
		processedHeader = true
	}
	return err
}

func getRoomsFromString(roomsString string, ctx context.Context, buildingsMap map[string]datastore.Key) []*datastore.Key {
	roomStrings := strings.Split(roomsString, ",")
	var rooms []*datastore.Key
	for _, r := range roomStrings {
		parts := strings.Split(r, "_")
		buildingKey := (buildingsMap[parts[0]])
		if len(parts) == 1 {
			q := datastore.NewQuery("Room").Filter("Building =", &buildingKey).KeysOnly()
			roomKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, nil)
			if err != nil {
				log.Printf("fetching rooms for building %s: %v", parts[0], err)
			}
			rooms = append(rooms, roomKeys...)
		}
		if len(parts) == 2 {
			roomNumber, _ := strconv.Atoi(parts[1])
			q := datastore.NewQuery("Room").Filter("Building =", &buildingKey).Filter("RoomNumber =", roomNumber).KeysOnly()
			roomKeys, err := dsclient.FromContext(ctx).GetAll(ctx, q, nil)
			if err != nil {
				log.Printf("fetching room %v %v: %v", parts[0], parts[1], err)
			}
			rooms = append(rooms, roomKeys...)
		}
	}
	return rooms

}

type ImportedGuest struct {
	GuestId       int
	FirstName     string
	LastName      string
	Nickname      string
	Email         string
	InviteeId     int
	HomePhone     string
	CellPhone     string
	AgeOverride   float64
	Birthdate     time.Time
	NeedBirthdate bool
	InviteCode    string
	Address       string
	Pronouns      person.PronounSet
}

func ImportGuests(w http.ResponseWriter, ctx context.Context) map[int]*datastore.Key {
	b := new(bytes.Buffer)
	guestFile, err := os.Open(Import_Data_Directory + "/" + Guest_Data_File_Name)
	if err != nil {
		log.Printf("File error: %v", err)
	}
	defer guestFile.Close()

	guestMap := make(map[int]*datastore.Key)

	scanner := bufio.NewScanner(guestFile)
	processedHeader := false
	for scanner.Scan() {
		var Guest ImportedGuest
		if processedHeader {
			guestRow := scanner.Text()
			fields := strings.Split(guestRow, "\t")
			guestIdInt, _ := strconv.Atoi(fields[0])
			Guest.GuestId = guestIdInt
			Guest.FirstName = fields[1]
			Guest.LastName = fields[2]
			Guest.Nickname = fields[3]
			Guest.Email = fields[4]
			Guest.InviteeId, _ = strconv.Atoi(fields[5])
			Guest.HomePhone = fields[6]
			Guest.CellPhone = fields[7]
			Guest.AgeOverride, _ = strconv.ParseFloat(fields[8], 64)

			layout := "2006-01-02 15:04:05"
			Guest.Birthdate, _ = time.Parse(layout, fields[9])
			Guest.NeedBirthdate = fields[10] == "1"
			Guest.InviteCode = fields[11]
			Guest.Address = strings.Replace(fields[12], "|", "\n", -1)
			pronoun := fields[13]
			switch pronoun {
			case "she":
				Guest.Pronouns = person.She
			case "he":
				Guest.Pronouns = person.He
			case "zie":
				Guest.Pronouns = person.Zie
			default:
				Guest.Pronouns = person.They
			}

			personKey, _ := CreatePersonFromImportedGuest(ctx, w, Guest)
			guestMap[guestIdInt] = personKey
		}
		processedHeader = true

	}

	if err := scanner.Err(); err != nil {
		log.Printf("GetAll: %v", err)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, b)
	return guestMap

}

func CreatePersonFromImportedGuest(ctx context.Context, w http.ResponseWriter, guest ImportedGuest) (*datastore.Key, error) {
	phone := guest.CellPhone
	if phone == "" {
		phone = guest.HomePhone
	}
	//clean phone number
	reg, _ := regexp.Compile(`[^\d]+`)

	phone = reg.ReplaceAllString(phone, "")

	p := person.Person{
		OldGuestId:    guest.GuestId,
		OldInviteeId:  guest.InviteeId,
		OldInviteCode: guest.InviteCode,
		FirstName:     guest.FirstName,
		LastName:      guest.LastName,
		Nickname:      guest.Nickname,
		Pronouns:      guest.Pronouns,
		Email:         guest.Email,
		Telephone:     phone,
		FallbackAge:   guest.AgeOverride,
		Birthdate:     guest.Birthdate,
		NeedBirthdate: guest.NeedBirthdate,
		Address:       guest.Address,
		LoginCode:     login.RandomLoginCodeString(),
	}

	w.Write([]byte(fmt.Sprintf("Adding person: %s\n", p.FullName())))
	key, err2 := dsclient.FromContext(ctx).Put(ctx, person.PersonKey(ctx), &p)
	if err2 != nil {
		log.Printf("%v", err2)
	}
	return key, err2
}

func ImportRsvps(w http.ResponseWriter, ctx context.Context, guestMap map[int]*datastore.Key) {
	b := new(bytes.Buffer)
	rsvpFile, err := os.Open(Import_Data_Directory + "/" + RSVP_Data_File_Name)
	if err != nil {
		log.Printf("%v", err)
	}
	defer rsvpFile.Close()

	e, err := event.GetAllEvents(ctx)
	if err != nil {
		log.Printf("%v", err)
	}
	// map from eventID to event
	eventMap := make(map[int]*event.Event)

	for i, event := range e {
		eventMap[event.EventId] = e[i]
	}

	var invitationCount [7]int

	scanner := bufio.NewScanner(rsvpFile)
	processedHeader := false

	for scanner.Scan() {
		if processedHeader {
			rsvpRow := scanner.Text()
			fields := strings.Split(rsvpRow, "\t")
			eventId, _ := strconv.Atoi(fields[0])
			guestIds := strings.Split(fields[1], ",")
			//names := strings.Split(fields[2], ",")
			rsvps := strings.Split(fields[3], ",")

			invitationCount[eventId]++

			var personKeys []*datastore.Key

			rsvpMap := make(map[*datastore.Key]invitation.RsvpStatus)

			for i, guestId := range guestIds {
				guestIdInt, _ := strconv.Atoi(guestId)
				personKey, exists := guestMap[guestIdInt]
				if !exists {
					log.Printf("Missing person in %s", fields[2])
					continue
				}
				var p person.Person
				dsclient.FromContext(ctx).Get(ctx, personKey, &p)
				personKeys = append(personKeys, personKey)

				rsvpChar := rsvps[i]
				if rsvpChar != "-" {
					rsvp := getRsvpStatusFromCode(eventId, rsvpChar)
					rsvpMap[personKey] = rsvp
				}
			}

			var invitation Invitation
			invitation.Event = eventMap[eventId].Key
			invitation.Invitees = personKeys
			invitation.RsvpMap = rsvpMap

			invitationKey := datastore.IncompleteKey("Invitation", nil)

			invitationKey, err = dsclient.FromContext(ctx).Put(ctx, invitationKey, &invitation)
			if err != nil {
				log.Printf("RSVPs: %v -- %s", err, rsvpRow)
			}

			w.Write([]byte(fmt.Sprintf("Adding retroactive invitation for %s (%v)\n", printInvitation(ctx, invitationKey, &invitation), *invitationKey)))

		}
		processedHeader = true
	}

	w.Write([]byte("\n"))
	for i, c := range invitationCount {
		if i > 0 {
			w.Write([]byte(fmt.Sprintf("%s: %d invitations\n", eventMap[i].ShortName, c)))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("%v", err)
		//log.Fatal(err)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, b)

}

func getRsvpStatusFromCode(eventId int, status string) invitation.RsvpStatus {
	switch status {
	case "n":
		return invitation.No
	case "m":
		return invitation.Maybe
	}

	switch eventId {
	case 1:
		switch status {
		case "y":
			return invitation.FriSat
		case "f":
			return invitation.Fri
		case "s":
			return invitation.Sat
		case "w":
			return invitation.WeddingOnly
		}
	case 2, 3:
		switch status {
		case "y":
			return invitation.FriSatSun
		case "f":
			return invitation.FriSat
		case "s":
			return invitation.SatSun
		}
	case 4:
		switch status {
		case "y":
			return invitation.FriSat
		case "f":
			return invitation.ThuFriSat
		case "s":
			return invitation.SatSun
		case "e":
			return invitation.FriSatPlusEither
		}
	case 5:
		switch status {
		case "y":
			return invitation.FriSat
		case "f":
			return invitation.ThuFriSat
		case "s":
			return invitation.SatSun
		}
	}

	return invitation.No
}

func ImportFoodPreferences(w http.ResponseWriter, ctx context.Context, guestMap map[int]*datastore.Key) {
	b := new(bytes.Buffer)

	allRestrictions := person.GetAllFoodRestrictionTags()

	foodFile, err := os.Open(Import_Data_Directory + "/" + Food_File_Name)
	if err != nil {
		log.Printf("%v", err)
	}
	defer foodFile.Close()

	scanner := bufio.NewScanner(foodFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			var restrictions []person.FoodRestriction
			foodRow := scanner.Text()
			fields := strings.Split(foodRow, "\t")
			guestIdInt, _ := strconv.Atoi(fields[0])
			name := fields[2]
			dietCode := fields[3]
			switch dietCode {
			case "v":
				restrictions = append(restrictions, person.Vegetarian)
			case "n":
				restrictions = append(restrictions, person.Vegan)
			case "f":
				restrictions = append(restrictions, person.VegetarianPlusFish)
			case "r":
				restrictions = append(restrictions, person.NoRedMeat)
			}

			if fields[4] == "1" {
				restrictions = append(restrictions, person.Kosher)
			}
			if fields[5] == "1" {
				restrictions = append(restrictions, person.NoDairy)
			}
			if fields[6] == "1" {
				restrictions = append(restrictions, person.NoGluten)
			}
			if fields[7] == "1" {
				restrictions = append(restrictions, person.DangerousAllergy)
			}
			if fields[8] == "1" {
				restrictions = append(restrictions, person.InconvenientAllergy)
			}

			foodIssues := ""
			foodNotes := strings.Replace(fields[9], "|", "\n", -1)

			personKey := guestMap[guestIdInt]
			var p person.Person
			err := dsclient.FromContext(ctx).Get(ctx, personKey, &p)
			if err != nil {
				log.Printf("%v: %v - %s", err, personKey.Encode(), foodRow)
			}
			p.FoodRestrictions = restrictions
			for _, rest := range restrictions {
				foodIssues += allRestrictions[rest].Description + ", "
			}

			p.FoodNotes = foodNotes

			w.Write([]byte(fmt.Sprintf("Restrictions for %s: %s %s\n", name, foodIssues, foodNotes)))
			_, err = dsclient.FromContext(ctx).Put(ctx, personKey, &p)
			if err != nil {
				log.Printf("%v", err)
			}
		}
		processedHeader = true
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, b)
}

func AskReloadHousingSetup(ctx context.Context, wr WrappedRequest) {
	wr.ResponseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(wr.ResponseWriter, `
	<form method="POST" action="/doReloadHousingSetup">
	<input type="submit" value="Do it">
	</form>
	`)
	//fmt.Fprintf(wr.ResponseWriter, "NO")
}

func ReloadHousingSetup(ctx context.Context, wr WrappedRequest) {
	ClearAllData(ctx, wr, []string{"Venue", "Building", "Room"})
	wr.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
	SetupVenues(wr.ResponseWriter, ctx)
	wr.ResponseWriter.Write([]byte("\n\n"))
	time.Sleep(2 * time.Second)
	SetupBuildings(wr.ResponseWriter, ctx)
	wr.ResponseWriter.Write([]byte("\n\n"))
	time.Sleep(2 * time.Second)
	SetupRooms(wr.ResponseWriter, ctx)
	wr.ResponseWriter.Write([]byte("\n\n"))
	time.Sleep(2 * time.Second)

	venuesMap := make(map[string]*datastore.Key)
	var venues []venue.Venue
	q := datastore.NewQuery("Venue")
	keys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &venues)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	for i, venueKey := range keys {
		venuesMap[(venues[i]).ShortName] = venueKey
	}

	buildingsMap := make(map[string]datastore.Key)
	var buildings []housing.Building
	q = datastore.NewQuery("Building")
	keys, err = dsclient.FromContext(ctx).GetAll(ctx, q, &buildings)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	for i, buildingKey := range keys {
		buildingsMap[(buildings[i]).Code] = *buildingKey
	}

	eventsMap := make(map[string]*event.Event)
	events, err := event.GetAllEvents(ctx)
	if err != nil {
		log.Printf("GetAllEvents: %v", err)
	}
	for _, ev := range events {
		eventsMap[ev.ShortName] = ev
	}

	eventsFile, err := os.Open(Import_Data_Directory + "/" + Events_Data_File_Name)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	defer eventsFile.Close()
	scanner := bufio.NewScanner(eventsFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			eventRow := scanner.Text()

			fields := strings.Split(eventRow, "\t")

			// Add venue to events
			venueKey := venuesMap[fields[3]]
			// Add rooms to events
			rooms := getRoomsFromString(fields[8], ctx, buildingsMap)

			ev := eventsMap[fields[2]]

			ev.SetVenueKey(venueKey)
			ev.Rooms = rooms

			err := event.PutEvent(ctx, ev)
			if err != nil {
				log.Printf("PutEvent: %v", err)
			}
		}
		processedHeader = true
	}
}

func SetupVenues(w http.ResponseWriter, ctx context.Context) error {
	venuesFile, err := os.Open(Import_Data_Directory + "/" + Venues_File_Name)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	defer venuesFile.Close()

	scanner := bufio.NewScanner(venuesFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			venueRow := scanner.Text()
			fields := strings.Split(venueRow, "\t")
			name := fields[0]
			shortName := fields[1]
			contactPerson := fields[2]
			contactEmail := fields[3]
			contactPhone := fields[4]
			website := fields[5]

			venue := venue.Venue{
				Name:          name,
				ShortName:     shortName,
				ContactPerson: contactPerson,
				ContactEmail:  contactEmail,
				ContactPhone:  contactPhone,
				Website:       website,
			}

			_, err := dsclient.FromContext(ctx).Put(ctx, datastore.IncompleteKey("Venue", nil), &venue)
			if err != nil {
				log.Printf("%v", err)
			}
			w.Write([]byte(fmt.Sprintf("Loading venue %s\n", fields[0])))
		}
		processedHeader = true
	}
	return err
}

func SetupBuildings(w http.ResponseWriter, ctx context.Context) error {
	buildingsFile, err := os.Open(Import_Data_Directory + "/" + Buildings_File_Name)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	defer buildingsFile.Close()

	venuesMap := make(map[string]*datastore.Key)
	var venues []venue.Venue
	q := datastore.NewQuery("Venue")
	keys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &venues)
	for i, venueKey := range keys {
		venuesMap[(venues[i]).ShortName] = venueKey
	}

	propertiesMap := make(map[string]int)
	for _, hpb := range GetAllHousingPreferenceBooleans() {
		propertiesMap[hpb.Name] = hpb.Bit
	}
	log.Printf("Properties Map: %v", propertiesMap)

	scanner := bufio.NewScanner(buildingsFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			buildingRow := scanner.Text()
			fields := strings.Split(buildingRow, "\t")
			venue := venuesMap[fields[0]]
			name := fields[1]
			code := fields[2]
			floorplanUrl := fields[3]
			propertyList := fields[4]
			propertyStrings := strings.Split(propertyList, ",")
			properties := 0
			for _, b := range propertyStrings {
				properties += propertiesMap[b]
				log.Printf("%s: %s --> %d", name, b, propertiesMap[b])
			}
			log.Printf("%s total properties: %d", name, properties)

			building := housing.Building{
				Venue:             venue,
				Name:              name,
				Code:              code,
				FloorplanImageUrl: floorplanUrl,
				Properties:        properties,
			}

			_, err := dsclient.FromContext(ctx).Put(ctx, datastore.IncompleteKey("Building", venue), &building)
			if err != nil {
				log.Printf("%v", err)
			}
			w.Write([]byte(fmt.Sprintf("Loading building %s\n", fields[1])))
		}
		processedHeader = true
	}
	return err
}

func SetupRooms(w http.ResponseWriter, ctx context.Context) error {
	roomsFile, err := os.Open(Import_Data_Directory + "/" + Rooms_File_Name)
	if err != nil {
		log.Printf("GetAll: %v", err)
	}
	defer roomsFile.Close()

	buildingsMap := make(map[string]*datastore.Key)
	var buildings []housing.Building
	q := datastore.NewQuery("Building")
	keys, err := dsclient.FromContext(ctx).GetAll(ctx, q, &buildings)
	for i, buildingKey := range keys {
		buildingsMap[(buildings[i]).Code] = buildingKey
	}

	propertiesMap := make(map[string]int)
	for _, hpb := range GetAllHousingPreferenceBooleans() {
		propertiesMap[hpb.Name] = int(hpb.Bit)
	}

	scanner := bufio.NewScanner(roomsFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			buildingRow := scanner.Text()
			fields := strings.Split(buildingRow, "\t")
			building := buildingsMap[fields[0]]
			number, _ := strconv.Atoi(fields[1])
			partition := fields[2]
			propertyList := fields[3]
			propertyStrings := strings.Split(propertyList, ",")
			properties := 0
			for _, b := range propertyStrings {
				properties += propertiesMap[b]
			}
			var bedSizes []housing.BedSize
			for _, c := range fields[4] {
				switch c {
				case 'K':
					bedSizes = append(bedSizes, housing.King)
				case 'Q':
					bedSizes = append(bedSizes, housing.Queen)
				case 'D':
					bedSizes = append(bedSizes, housing.Double)
				case 'T':
					bedSizes = append(bedSizes, housing.Twin)
				case 'C':
					bedSizes = append(bedSizes, housing.Cot)
				}
			}

			top, _ := strconv.Atoi(fields[5])
			left, _ := strconv.Atoi(fields[6])
			width, _ := strconv.Atoi(fields[7])
			height, _ := strconv.Atoi(fields[8])

			room := housing.Room{
				Building:    building,
				RoomNumber:  number,
				Partition:   partition,
				Properties:  properties,
				Beds:        bedSizes,
				ImageTop:    top,
				ImageLeft:   left,
				ImageWidth:  width,
				ImageHeight: height,
			}

			_, err := dsclient.FromContext(ctx).Put(ctx, datastore.IncompleteKey("Room", building), &room)
			if err != nil {
				log.Printf("%v", err)
			}
			w.Write([]byte(fmt.Sprintf("Loading room %s%s%s\n", fields[0], fields[1], fields[2])))
		}
		processedHeader = true
	}
	return err
}
