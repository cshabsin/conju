package conju

import (
	"context"
	"google.golang.org/appengine/datastore"
)

type Venue struct {
	Name          string
	ShortName     string
	ContactPerson string
	ContactPhone  string
	ContactEmail  string
	Website       string
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
	Room     Room
	Building Building
}

func (room Room) makeRealRoom(ctx context.Context) RealRoom {
	var building Building
	datastore.Get(ctx, room.Building, building)

	var realRoom RealRoom
	realRoom.Building = building
	realRoom.Room = room

	return realRoom
}

func (room RealRoom) AllProperties() int {
	return room.Building.Properties | room.Room.Properties
}

/*
(room *Room) String() {
	toReturn := ""
	if room.Building != nil {
	   var building Building
	   datastore.Get(ctx
	   toReturn
	}

	if Description != "" {
	   toReturn += Description
	}
} */
