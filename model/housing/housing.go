package housing

import (
	"context"

	"google.golang.org/appengine/datastore"
)

type venueDB struct {
	Name          string
	ShortName     string
	ContactPerson string
	ContactPhone  string
	ContactEmail  string
	Website       string
}

type Venue struct {
	Key           *datastore.Key
	Name          string
	ShortName     string
	ContactPerson string
	ContactPhone  string
	ContactEmail  string
	Website       string
}

func AllVenues(ctx context.Context) ([]*Venue, error) {
	q := datastore.NewQuery("Venue")
	var allVenues []*venueDB
	venueKeys, err := q.GetAll(ctx, &allVenues)
	if err != nil {
		return nil, err
	}
	var venues []*Venue
	for i, v := range allVenues {
		venues = append(venues, &Venue{
			Key:           venueKeys[i],
			Name:          v.Name,
			ShortName:     v.ShortName,
			ContactPerson: v.ContactPerson,
			ContactPhone:  v.ContactPhone,
			ContactEmail:  v.ContactEmail,
			Website:       v.Website,
		})
	}
	return venues, nil
}

type Building struct {
	Venue             *datastore.Key
	Name              string
	Code              string
	Properties        int
	FloorplanImageUrl string
}

type BedSize int

const (
	King = iota
	Queen
	Double
	Twin
	Cot
)

type roomDB struct {
	Building    *datastore.Key
	RoomNumber  int
	Description string
	Partition   string
	Properties  int
	Beds        []BedSize

	ImageTop    int
	ImageLeft   int
	ImageWidth  int
	ImageHeight int
}

type RealRoom struct {
	Room       *roomDB
	Realized   bool
	Building   *Building
	BedsString string
}

func (room *RealRoom) Realize(ctx context.Context) error {
	if room.BedsString == "" {
		var bedstring string
		for _, bed := range room.Room.Beds {
			switch bed {
			case King:
				bedstring += "K"
			case Queen:
				bedstring += "Q"
			case Double:
				bedstring += "D"
			case Twin:
				bedstring += "T"
			case Cot:
				bedstring += "C"
			}
		}
		room.BedsString = bedstring
	}
	//lint:ignore SA9003 empty branch will be filled later
	if room.Building == nil {
		// TODO(cshabsin): fetch the building from the room.
	}
	return nil
}

func (room *RealRoom) AllProperties(ctx context.Context) (int, error) {
	if err := room.Realize(ctx); err != nil {
		return 0, err
	}
	return room.Building.Properties | room.Room.Properties, nil
}
