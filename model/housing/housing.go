package housing

import (
	"cloud.google.com/go/datastore"
)

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

type Room struct {
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
	Room       Room
	Building   Building
	BedsString string
}

func (room RealRoom) AllProperties() int {
	return room.Building.Properties | room.Room.Properties
}
