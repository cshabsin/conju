package conju

import (
	"bufio"
	"bytes"
	"context"
	//	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine/datastore"
)

//const Import_Data_Directory = "test_import_data"
const Import_Data_Directory = "real_import_data"

const Guest_Data_File_Name = "Guests_to_Import.tsv"
const RSVP_Data_File_Name = "rsvps.tsv"
const Events_Data_File_Name = "events.tsv"

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

func ReloadData(wr WrappedRequest) {
	// TODO: print out report of what got imported
	ClearAllData(wr)
	SetupEvents(wr.ResponseWriter, wr.Context)
	ImportGuests(wr.ResponseWriter, wr.Context)
}

func SetupEvents(w http.ResponseWriter, ctx context.Context) error {
	eventsFile, err := os.Open(Import_Data_Directory + "/" + Events_Data_File_Name)
	if err != nil {
		log.Fatal(err)
	}
	defer eventsFile.Close()

	layout := "1/2/2006"
	scanner := bufio.NewScanner(eventsFile)
	processedHeader := false
	for scanner.Scan() {
		if processedHeader {
			eventRow := scanner.Text()

			fields := strings.Split(eventRow, "\t")
			startDate, _ := time.Parse(layout, fields[3])
			endDate, _ := time.Parse(layout, fields[4])
			eventId, _ := strconv.Atoi(fields[0])
			eventKey, _ := CreateEvent(ctx, eventId, fields[1], fields[2], startDate, endDate,
				fields[5] == "1")
			if fields[5] == "1" {
				ce_key := datastore.NewKey(ctx, "CurrentEvent", "current_event", 0, nil)
				ce := CurrentEvent{eventKey}
				datastore.Put(ctx, ce_key, &ce)
			}

			//var thisEvent Event
			//_ = datastore.Get(ctx, eventKey, thisEvent)
			//w.Write([]byte(fmt.Sprintf("Loaded %s event %s (%s) %s - %s\n", eventKey.Encode(), thisEvent.Name, thisEvent.ShortName, thisEvent.StartDate.Format("01/02/2006"), thisEvent.EndDate.Format("01/02/2006"))))
		}
		processedHeader = true
	}
	return err
}

func ImportGuests(w http.ResponseWriter, ctx context.Context) {
	b := new(bytes.Buffer)
	guestFile, err := os.Open(Import_Data_Directory + "/" + Guest_Data_File_Name)
	if err != nil {
		log.Fatal(err)
	}
	defer guestFile.Close()

	scanner := bufio.NewScanner(guestFile)
	processedHeader := false
	for scanner.Scan() {
		var Guest ImportedGuest
		if processedHeader {
			guestRow := scanner.Text()
			fields := strings.Split(guestRow, "\t")
			Guest.GuestId, err = strconv.Atoi(fields[0])
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

			CreatePersonFromImportedGuest(ctx, Guest)

		}
		processedHeader = true

	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.Copy(w, b)

}

func CreatePersonFromImportedGuest(ctx context.Context, guest ImportedGuest) error {
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

	_, err2 := datastore.Put(ctx, PersonKey(ctx), &p)
	return err2
}
