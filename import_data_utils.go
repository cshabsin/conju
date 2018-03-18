package conju

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	//"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

//const Import_Data_Directory = "test_import_data"
const Import_Data_Directory = "real_import_data"

const Guest_Data_File_Name = "Guests_to_Import.tsv"
const RSVP_Data_File_Name = "rsvps.tsv"
const Events_Data_File_Name = "events.tsv"
const Food_File_Name = "food.tsv"

func ReloadData(wr WrappedRequest) {
	// TODO: print out report of what got imported
	ClearAllData(wr)
	wr.ResponseWriter.Write([]byte("\n\n"))
	SetupEvents(wr.ResponseWriter, wr.Context)
	wr.ResponseWriter.Write([]byte("\n\n"))
	guestMap := ImportGuests(wr.ResponseWriter, wr.Context)
	wr.ResponseWriter.Write([]byte("\n\n"))
	ImportRsvps(wr.ResponseWriter, wr.Context, guestMap)
	wr.ResponseWriter.Write([]byte("\n\n"))
	ImportFoodPreferences(wr.ResponseWriter, wr.Context, guestMap)
}

func SetupEvents(w http.ResponseWriter, ctx context.Context) error {
	eventsFile, err := os.Open(Import_Data_Directory + "/" + Events_Data_File_Name)
	if err != nil {
		log.Errorf(ctx, "GetAll: %v", err)
	}
	defer eventsFile.Close()

	layout := "1/2/2006"

	rsvpStatusMap := make(map[string]RsvpStatus)
	allRsvpStatuses := GetAllRsvpStatuses()
	for _, status := range allRsvpStatuses {
		rsvpStatusMap[status.ShortDescription] = status.Status
	}

	scanner := bufio.NewScanner(eventsFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			eventRow := scanner.Text()

			fields := strings.Split(eventRow, "\t")
			startDate, _ := time.Parse(layout, fields[3])
			endDate, _ := time.Parse(layout, fields[4])
			eventId, _ := strconv.Atoi(fields[0])
			rsvpStatusStrings := strings.Split(fields[6], ",")
			var rsvpStatuses []RsvpStatus
			for _, rsvpStatusString := range rsvpStatusStrings {
				rsvpStatuses = append(rsvpStatuses, rsvpStatusMap[rsvpStatusString])
			}

			_, _ = CreateEvent(ctx, eventId, fields[1], fields[2], startDate, endDate, rsvpStatuses,
				fields[5] == "1")
			w.Write([]byte(fmt.Sprintf("Loading event %s (%s) %s - %s\n", fields[1], fields[2], startDate.Format("01/02/2006"), endDate.Format("01/02/2006"))))
		}
		processedHeader = true
	}
	return err
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
	Pronouns      PronounSet
}

func ImportGuests(w http.ResponseWriter, ctx context.Context) map[int]datastore.Key {
	b := new(bytes.Buffer)
	guestFile, err := os.Open(Import_Data_Directory + "/" + Guest_Data_File_Name)
	if err != nil {
		log.Errorf(ctx, "File error: %v", err)
	}
	defer guestFile.Close()

	guestMap := make(map[int]datastore.Key)

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
			Guest.InviteeId, err = strconv.Atoi(fields[5])
			Guest.HomePhone = fields[6]
			Guest.CellPhone = fields[7]
			Guest.AgeOverride, err = strconv.ParseFloat(fields[8], 64)

			layout := "2006-01-02 15:04:05"
			Guest.Birthdate, err = time.Parse(layout, fields[9])
			Guest.NeedBirthdate = fields[10] == "1"
			Guest.InviteCode = fields[11]
			Guest.Address = strings.Replace(fields[12], "|", "\n", -1)
			pronoun := fields[13]
			switch pronoun {
			case "she":
				Guest.Pronouns = She
			case "he":
				Guest.Pronouns = He
			case "zie":
				Guest.Pronouns = Zie
			default:
				Guest.Pronouns = They
			}

			personKey, _ := CreatePersonFromImportedGuest(ctx, w, Guest)
			guestMap[guestIdInt] = personKey
		}
		processedHeader = true

	}

	if err := scanner.Err(); err != nil {
		log.Errorf(ctx, "GetAll: %v", err)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, b)
	return guestMap

}

func CreatePersonFromImportedGuest(ctx context.Context, w http.ResponseWriter, guest ImportedGuest) (datastore.Key, error) {
	phone := guest.CellPhone
	if phone == "" {
		phone = guest.HomePhone
	}
	//clean phone number
	reg, _ := regexp.Compile("[^\\d]+")

	phone = reg.ReplaceAllString(phone, "")

	p := Person{
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
	}

	w.Write([]byte(fmt.Sprintf("Adding person: %s\n", p.FullName())))
	key, err2 := datastore.Put(ctx, PersonKey(ctx), &p)
	if err2 != nil {
		log.Errorf(ctx, "%v", err2)
	}
	return *key, err2
}

func ImportRsvps(w http.ResponseWriter, ctx context.Context, guestMap map[int]datastore.Key) {
	b := new(bytes.Buffer)
	rsvpFile, err := os.Open(Import_Data_Directory + "/" + RSVP_Data_File_Name)
	if err != nil {
		log.Errorf(ctx, "%v", err)
	}
	defer rsvpFile.Close()

	q := datastore.NewQuery("Event")
	var e []*Event
	eventKeys, err := q.GetAll(ctx, &e)
	eventKeyMap := make(map[int]datastore.Key)
	eventMap := make(map[int]Event)

	for i, event := range e {
		eventKeyMap[event.EventId] = *eventKeys[i]
		eventMap[event.EventId] = *e[i]
	}

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

			if err != nil {
				log.Errorf(ctx, "%v", err)
			}

			eventKey := eventKeyMap[eventId]

			var invitees []Person
			var personKeys []*datastore.Key

			rsvpMap := make(map[*datastore.Key]RsvpStatus)

			for i, guestId := range guestIds {
				guestIdInt, _ := strconv.Atoi(guestId)
				personKey := guestMap[guestIdInt]
				var p Person
				datastore.Get(ctx, &personKey, &p)
				invitees = append(invitees, p)
				personKeys = append(personKeys, &personKey)

				rsvpChar := rsvps[i]
				if rsvpChar != "-" {
					rsvp := getRsvpStatusFromCode(eventId, rsvpChar)
					rsvpMap[&personKey] = rsvp
				}
			}

			var invitation Invitation
			invitation.Event = &eventKey
			invitation.Invitees = personKeys
			invitation.RsvpMap = rsvpMap

			invitationKey := datastore.NewIncompleteKey(ctx, "Invitation", nil)

			_, err = datastore.Put(ctx, invitationKey, &invitation)
			if err != nil {
				log.Errorf(ctx, "------ %v", err)
			}

			w.Write([]byte(fmt.Sprintf("Adding retroactive invitation for %s: %s\n", eventMap[eventId].ShortName, CollectiveAddress(invitees, Informal))))

		}
		processedHeader = true
	}

	if err := scanner.Err(); err != nil {
		//log.Fatal(err)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, b)

}

func getRsvpStatusFromCode(eventId int, status string) RsvpStatus {

	switch status {
	case "n":
		return No
	case "m":
		return Maybe
	}

	switch eventId {
	case 1:
		switch status {
		case "y":
			return FriSat
		case "f":
			return Fri
		case "s":
			return Sat
		case "w":
			return WeddingOnly
		}
	case 2, 3:
		switch status {
		case "y":
			return FriSatSun
		case "f":
			return FriSat
		case "s":
			return SatSun
		}
	case 4:
		switch status {
		case "y":
			return FriSat
		case "f":
			return ThuFriSat
		case "s":
			return SatSun
		case "e":
			return FriSatPlusEither
		}
	case 5:
		switch status {
		case "y":
			return FriSat
		case "f":
			return ThuFriSat
		case "s":
			return SatSun
		}
	}

	return No
}

func ImportFoodPreferences(w http.ResponseWriter, ctx context.Context, guestMap map[int]datastore.Key) {
	b := new(bytes.Buffer)

	allRestrictions := GetAllFoodRestrictionTags()

	foodFile, err := os.Open(Import_Data_Directory + "/" + Food_File_Name)
	if err != nil {
		log.Errorf(ctx, "%v", err)
	}
	defer foodFile.Close()

	scanner := bufio.NewScanner(foodFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			var restrictions []FoodRestriction
			foodRow := scanner.Text()
			fields := strings.Split(foodRow, "\t")
			guestIdInt, _ := strconv.Atoi(fields[0])
			name := fields[2]
			dietCode := fields[3]
			switch dietCode {
			case "v":
				restrictions = append(restrictions, Vegetarian)
			case "n":
				restrictions = append(restrictions, Vegan)
			case "f":
				restrictions = append(restrictions, VegetarianPlusFish)
			case "r":
				restrictions = append(restrictions, NoRedMeat)
			}

			if fields[4] == "1" {
				restrictions = append(restrictions, Kosher)
			}
			if fields[5] == "1" {
				restrictions = append(restrictions, NoDairy)
			}
			if fields[6] == "1" {
				restrictions = append(restrictions, NoGluten)
			}
			if fields[7] == "1" {
				restrictions = append(restrictions, DangerousAllergy)
			}
			if fields[8] == "1" {
				restrictions = append(restrictions, InconvenientAllergy)
			}

			foodIssues := ""
			foodNotes := strings.Replace(fields[9], "|", "\n", -1)

			personKey := guestMap[guestIdInt]
			var p Person
			err := datastore.Get(ctx, &personKey, &p)
			if err != nil {
				log.Errorf(ctx, "%v: %v - %s", err, personKey.Encode(), foodRow)
			}
			p.FoodRestrictions = restrictions
			for _, rest := range restrictions {
				foodIssues += allRestrictions[rest].Description + ", "
			}

			p.FoodNotes = foodNotes

			w.Write([]byte(fmt.Sprintf("Restrictions for %s: %s %s\n", name, foodIssues, foodNotes)))
			_, err = datastore.Put(ctx, &personKey, &p)
			if err != nil {
				log.Errorf(ctx, "%v", err)
			}
		}
		processedHeader = true
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, b)
}
