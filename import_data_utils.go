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

	"google.golang.org/appengine/datastore"
)

const Guest_Data_File_Name = "test_import_data/Test_Guests.tsv"

//const Guest_Data_File_Name = "Guests_to_Import.tsv"

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

func ImportData(wr WrappedRequest) {
	ImportGuests(wr.ResponseWriter, wr.Request, wr.Context)
}

func ImportGuests(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	b := new(bytes.Buffer)
	guestFile, err := os.Open(Guest_Data_File_Name)
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
			fmt.Fprintf(b, "%s\n", guestRow)
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
			fmt.Println(err)

			CreatePersonFromImportedGuest(ctx, Guest)

		}
		processedHeader = true
		fmt.Println()
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
