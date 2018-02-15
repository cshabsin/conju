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
    "strconv"
    "strings"
    "regexp"
    "time"
	
    "google.golang.org/appengine/datastore"
    "google.golang.org/appengine"
)

const Guest_Data_File_Name = "test_import_data/Test_Guests.tsv"

type ImportedGuest struct {
     GuestId int
     FirstName string
     LastName string
     Nickname string
     Email string
     InviteeId int
     HomePhone string
     CellPhone string
     AgeOverride float64
     Birthdate time.Time
     InviteCode string
     Address string
}


func ImportData(w http.ResponseWriter, r *http.Request) {
     ImportGuests(w, r)
}

func ImportGuests(w http.ResponseWriter, r *http.Request) {
    ctx := appengine.NewContext(r)
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
           fmt.Fprintf(b, "%d fields: %s\n", len(fields),guestRow)
	   Guest.GuestId, err = strconv.Atoi(fields[0])
       	   Guest.FirstName = fields[1]
	   Guest.LastName = fields[2]
	   Guest.Nickname = fields[3]
	   Guest.Email = fields[4]
	   Guest.InviteeId, err = strconv.Atoi(fields[5])
	   Guest.HomePhone = fields[6]
	   Guest.CellPhone = fields[7]
	   Guest.AgeOverride, err = strconv.ParseFloat(fields[8], 64)

	   layout := "2017-08-24 21:14:00"
	   Guest.Birthdate, err	= time.Parse(layout, fields[9]) 
	   Guest.InviteCode = fields[10]
	   Guest.Address = strings.Replace(fields[11],"|","\n",-1)

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



     p := Person {
       OldGuestId: guest.GuestId,
       OldInviteeId: guest.InviteeId,
       OldInviteCode: guest.InviteCode,
       FirstName: guest.FirstName,
       LastName:  guest.LastName,
       Nickname: guest.Nickname,
       Email: guest.Email,
       Telephone: phone,
       FallbackAge: guest.AgeOverride,
       Birthdate: guest.Birthdate,
       Address: guest.Address,
     }
     
	_, err2 := datastore.Put(ctx, PersonKey(ctx), &p)
	return err2
}
